//go:build e2e

package e2e

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// TestTheMailSettingsChangeOnlyTheLastLayerAndReachCron — two projects that
// differ only in where mail goes must share every php image layer but the last,
// and each PHP must send where its own project says, cron jobs included.
//
// sendmail_path used to be written in the middle of the image (php/ini). A
// muted clone sends to its own envelope address, so its image matched the
// template's up to that line and rebuilt the 14 layers after it: 6-12 s and
// ~2 GiB at peak per copy, measured on 2026-11-20. 4.5.68 moved the value into
// the container's environment, and cron jobs — started by cron with an
// environment of its own — got an empty sendmail_path and mail() returned
// false. Now the value is in php.ini again, as the image's last step.
//
// The witness for "one image" is the layer list, built without the CI cache:
// the question is what two builds on one machine share. The witness for "its
// own mail" is what PHP itself reads — the CLI's ini_get and php-fpm's own -i —
// and a message that arrives in mailpit, once from docker exec and once from a
// cron job.
func TestTheMailSettingsChangeOnlyTheLastLayerAndReachCron(t *testing.T) {
	install := newInstallation(t)
	run := func(p *project, timeout time.Duration, args ...string) string {
		t.Helper()
		out, err := p.tryRunWith(timeout, "", []string{"MADOCK_BUILD_CACHE="}, args...)
		if err != nil {
			t.Fatalf("madock %s failed: %v\n%s", strings.Join(args, " "), err, out)
		}
		return out
	}

	first := install.project("e2emailone")
	run(first, 15*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=php",
		"--search-engine=none",
		"--hosts=e2emailone.test",
	)

	second := install.project("e2emailtwo")
	run(second, 15*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=php",
		"--search-engine=none",
		"--hosts=e2emailtwo.test",
	)
	run(second, 2*time.Minute, "config:set", "-n", "php/sendmail/host", "-v", "relay.example.com")
	run(second, 2*time.Minute, "config:set", "-n", "php/sendmail/port", "-v", "587")
	run(second, 2*time.Minute, "config:set", "-n", "php/sendmail/from", "-v", "shop@example.com")

	install.resetProxy(first)
	run(first, 25*time.Minute, "start")
	run(second, 25*time.Minute, "start")

	a := imageLayers(t, "madock_e2emailone-php")
	b := imageLayers(t, "madock_e2emailtwo-php")
	requireAllButTheLastLayerShared(t, "the two projects' php images", a, b)

	cliPath := func(p *project) string {
		return strings.TrimSpace(run(p, 2*time.Minute, "cli", "php", "-r", `echo ini_get("sendmail_path");`))
	}
	fpmPath := func(p *project) string {
		out := run(p, 2*time.Minute, "cli", "sh", "-c", `for f in /usr/sbin/php-fpm*; do "$f" -i; done | grep '^sendmail_path'`)
		return strings.TrimSpace(out)
	}

	const firstWant = "/usr/bin/msmtp -t --port=1025 --host=host.docker.internal"
	const secondWant = "/usr/bin/msmtp -t --port=587 --host=relay.example.com --from=shop@example.com"

	if got := cliPath(first); got != firstWant {
		t.Errorf("the first project's PHP CLI reads sendmail_path %q, want %q", got, firstWant)
	}
	if got := cliPath(second); got != secondWant {
		t.Errorf("the second project's PHP CLI reads sendmail_path %q, want %q", got, secondWant)
	}
	// php-fpm -i prints "sendmail_path => <local> => <master>".
	if got := fpmPath(second); !strings.Contains(got, "=> "+secondWant+" =>") {
		t.Errorf("the second project's php-fpm reads %q, want %q", got, secondWant)
	}

	// The value is right; now the mail. The first project has the defaults, so
	// mail() hands the message to msmtp, which hands it to the proxy's mailpit
	// through host.docker.internal:1025 — the path every developer's mail
	// takes. Sent with a fifth argument, as Magento and Laravel do, so the -f
	// PHP appends to sendmail_path is part of what is checked.
	subject := fmt.Sprintf("e2e-mailpit-%d", time.Now().UnixNano())
	sent := run(first, 2*time.Minute, "cli", "php", "-r",
		fmt.Sprintf(`var_dump(mail("someone@e2emailone.test", %q, "body", "", "-fsender@e2emailone.test"));`, subject))
	if !strings.Contains(sent, "bool(true)") {
		t.Errorf("mail() from docker exec did not succeed:\n%s", sent)
	}
	if !mailpitHas(t, subject, 20*time.Second) {
		t.Errorf("mailpit has no message with subject %s", subject)
	}

	// The same from a cron job, which is where Magento sends its queued mail.
	// cron does not hand the container's environment to its jobs, so a value
	// kept there reached docker exec and php-fpm above and nothing here.
	cronSubject := fmt.Sprintf("e2e-cron-%d", time.Now().UnixNano())
	script := fmt.Sprintf(`<?php
file_put_contents("/tmp/e2e-cronmail.out", ini_get("sendmail_path") . "\n" . var_export(mail("someone@e2emailone.test", %q, "body", "", "-fsender@e2emailone.test"), true) . "\n");
`, cronSubject)
	run(first, 2*time.Minute, "cli", "php", "-r",
		fmt.Sprintf(`file_put_contents("/tmp/e2e-cronmail.php", base64_decode(%q));`, base64.StdEncoding.EncodeToString([]byte(script))))
	run(first, 2*time.Minute, "cli", "sh", "-c",
		`( crontab -l 2>/dev/null; echo '* * * * * php /tmp/e2e-cronmail.php' ) | crontab -`)
	run(first, 2*time.Minute, "cron:enable")
	if !mailpitHas(t, cronSubject, 150*time.Second) {
		out, _ := first.tryRunWith(time.Minute, "", nil, "cli", "sh", "-c", "cat /tmp/e2e-cronmail.out 2>&1; crontab -l")
		t.Errorf("mailpit has no message sent from cron (subject %s); what the job saw:\n%s", cronSubject, out)
	}
	run(first, 2*time.Minute, "cron:disable")

	// A change of mail settings rebuilds the last layer and nothing before it.
	run(first, 2*time.Minute, "config:set", "-n", "php/sendmail/from", "-v", "noreply@e2emailone.test")
	run(first, 25*time.Minute, "rebuild")
	if got := cliPath(first); got != firstWant+" --from=noreply@e2emailone.test" {
		t.Errorf("after config:set and rebuild the first project's PHP reads %q", got)
	}
	requireAllButTheLastLayerShared(t, "the php image before and after a new sender", a, imageLayers(t, "madock_e2emailone-php"))
}

// requireAllButTheLastLayerShared fails unless the two images have the same
// number of layers and differ in the last one at most.
func requireAllButTheLastLayerShared(t *testing.T, what string, a, b []string) {
	t.Helper()

	shared := 0
	for shared < len(a) && shared < len(b) && a[shared] == b[shared] {
		shared++
	}
	if len(a) != len(b) || shared < len(a)-1 {
		t.Errorf("%s: %d and %d layers, shared prefix %d — the mail settings reach more than the last layer", what, len(a), len(b), shared)
	}
}

// mailpitHas asks the proxy's mailpit, through its API on the loopback port
// the proxy publishes, whether a message with this subject arrived within the
// given time. msmtp returns once mailpit has accepted the message, but storing
// it is mailpit's own business; a cron job runs only on the next minute.
func mailpitHas(t *testing.T, subject string, within time.Duration) bool {
	t.Helper()

	deadline := time.Now().Add(within)
	for time.Now().Before(deadline) {
		resp, err := http.Get("http://127.0.0.1:8025/api/v1/messages?limit=200")
		if err == nil {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if strings.Contains(string(body), subject) {
				return true
			}
		}
		time.Sleep(2 * time.Second)
	}
	return false
}
