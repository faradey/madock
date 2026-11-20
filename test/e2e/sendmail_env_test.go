//go:build e2e

package e2e

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// TestPhpImageIsTheSameWhateverTheMailSettings — two projects that differ only
// in where mail goes must build one php image, and each PHP must still send
// where its own project says.
//
// sendmail_path used to be written into php.ini with the host, port and sender
// in it. A muted clone sends to its own envelope address, so its image matched
// the template's up to that line and rebuilt everything after it: 6-12 s and
// ~2 GiB at peak per copy, measured on 2026-11-20, and no two clones shared
// those layers. Now php.ini reads MADOCK_SENDMAIL_PATH and the compose file
// carries the value.
//
// The witness for "one image" is the layer list, as in the timezone test, and
// built without the CI cache for the same reason: the question is what two
// builds on one machine share. The witness for "its own mail" is what PHP
// itself reads — the CLI's ini_get and php-fpm's own -i, which parses the fpm
// php.ini with the container's environment the way the fpm master does.
func TestPhpImageIsTheSameWhateverTheMailSettings(t *testing.T) {
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
	if strings.Join(a, " ") != strings.Join(b, " ") {
		shared := 0
		for shared < len(a) && shared < len(b) && a[shared] == b[shared] {
			shared++
		}
		t.Errorf("the php images differ: %d and %d layers, shared prefix %d — the mail settings still reach the image", len(a), len(b), shared)
	}

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
		t.Errorf("mail() through the environment's sendmail_path did not succeed:\n%s", sent)
	}
	if !mailpitHas(t, subject) {
		t.Errorf("mailpit has no message with subject %s", subject)
	}

	// Mail settings change without a new image: the value is the container's.
	run(first, 2*time.Minute, "config:set", "-n", "php/sendmail/from", "-v", "noreply@e2emailone.test")
	run(first, 25*time.Minute, "start")
	if got := cliPath(first); got != firstWant+" --from=noreply@e2emailone.test" {
		t.Errorf("after config:set and start the first project's PHP reads %q", got)
	}
	if after := imageLayers(t, "madock_e2emailone-php"); strings.Join(after, " ") != strings.Join(a, " ") {
		t.Errorf("changing the sender produced a new php image")
	}
}

// mailpitHas asks the proxy's mailpit, through its API on the loopback port
// the proxy publishes, whether a message with this subject arrived. msmtp
// returns once mailpit has accepted the message, but storing it is mailpit's
// own business, so a few seconds are given.
func mailpitHas(t *testing.T, subject string) bool {
	t.Helper()

	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get("http://127.0.0.1:8025/api/v1/messages?limit=200")
		if err == nil {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if strings.Contains(string(body), subject) {
				return true
			}
		}
		time.Sleep(time.Second)
	}
	return false
}
