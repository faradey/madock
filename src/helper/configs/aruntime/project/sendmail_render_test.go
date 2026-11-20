package project

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/faradey/madock/v4/src/helper/testenv"
)

// renderSendmail renders a project and returns its php.Dockerfile and compose
// file. The project name is the same in every call: the comparison below is
// about the mail settings, and a different name would differ for its own reasons.
func renderSendmail(t *testing.T, overrides map[string]string) (dockerfile, compose string) {
	t.Helper()

	projectName := "mailimage"
	env := testenv.SetupWith(t, projectName, "mailimage.test", overrides)

	MakeConf(projectName)

	rendered := testenv.Collect(t, filepath.Join(env.ExecDir, "aruntime", "projects", projectName), env)
	for name, body := range rendered {
		switch {
		case strings.HasSuffix(name, "ctx/php.Dockerfile"):
			dockerfile = body
		case strings.HasSuffix(name, "docker-compose.yml") && !strings.Contains(name, "snapshot"):
			compose = body
		}
	}
	if dockerfile == "" || compose == "" {
		t.Fatalf("MakeConf produced no php.Dockerfile or docker-compose.yml (have %d files)", len(rendered))
	}

	return dockerfile, compose
}

// The php image must not depend on where mail goes. It used to: sendmail_path
// was written into php.ini with the host, port and sender in it, and a muted
// clone — which sends to its own envelope address — rebuilt every layer from
// that line on, 6-12 s and ~2 GiB per copy (measured 2026-11-20), sharing none
// of them with the template or with the next clone. Now the line reads
// MADOCK_SENDMAIL_PATH and the compose file carries the value.
func TestPhpImageDoesNotDependOnTheMailSettings(t *testing.T) {
	configs := map[string]map[string]string{
		"mail off": {
			"php/enabled":          "true",
			"php/sendmail/enabled": "false",
		},
		"mailpit": {
			"php/enabled":          "true",
			"php/sendmail/enabled": "true",
		},
		"relay with a sender": {
			"php/enabled":          "true",
			"php/sendmail/enabled": "true",
			"php/sendmail/host":    "relay.example.com",
			"php/sendmail/port":    "587",
			"php/sendmail/from":    "shop@example.com",
		},
		"muted clone": {
			"php/enabled": "true",
			"clone/muted": "true",
		},
	}
	want := map[string]string{
		"mail off":            `"MADOCK_SENDMAIL_PATH=/usr/sbin/sendmail -t -i"`,
		"mailpit":             `"MADOCK_SENDMAIL_PATH=/usr/bin/msmtp -t --port=1025 --host=host.docker.internal"`,
		"relay with a sender": `"MADOCK_SENDMAIL_PATH=/usr/bin/msmtp -t --port=587 --host=relay.example.com --from=shop@example.com"`,
		"muted clone":         `"MADOCK_SENDMAIL_PATH=/usr/bin/msmtp -t --port=1025 --host=host.docker.internal"`,
	}

	var first, firstName string
	for name, overrides := range configs {
		dockerfile, compose := renderSendmail(t, overrides)

		if !strings.Contains(dockerfile, `sendmail_path = "${MADOCK_SENDMAIL_PATH}"`) {
			t.Errorf("%s: php.ini is not pointed at the environment:\n%s", name, dockerfile)
		}
		// msmtp itself is installed in every image; its arguments must not be.
		for _, arg := range []string{"--host=", "--port=", "--from="} {
			if strings.Contains(dockerfile, arg) {
				t.Errorf("%s: %s reached the image", name, arg)
			}
		}
		if !strings.Contains(compose, want[name]) {
			t.Errorf("%s: compose file lacks %s", name, want[name])
		}

		if first == "" {
			first, firstName = dockerfile, name
			continue
		}
		if dockerfile != first {
			t.Errorf("php.Dockerfile for %q differs from %q — the image depends on the mail settings", name, firstName)
		}
	}
}
