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

// withoutSendmail drops the lines that carry the sendmail_path value, leaving
// what must be the same for every project built from one template.
func withoutSendmail(dockerfile string) string {
	var kept []string
	for _, line := range strings.Split(dockerfile, "\n") {
		if strings.Contains(line, `sendmail_path = "`) {
			continue
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}

// layerAfterSendmail returns the first instruction after the sendmail RUN that
// makes a layer of its own, or "" when only metadata follows. Docker rebuilds
// from the first line that differs, so anything here is rebuilt by every
// project whose mail settings differ from the template's.
func layerAfterSendmail(dockerfile string) string {
	lines := strings.Split(dockerfile, "\n")
	at := -1
	for i, line := range lines {
		if strings.HasPrefix(line, "RUN ") && strings.Contains(line, `sendmail_path = "`) {
			at = i
		}
	}
	if at < 0 {
		return "no sendmail RUN line at all"
	}
	for _, line := range lines[at+1:] {
		for _, instruction := range []string{"RUN ", "COPY ", "ADD "} {
			if strings.HasPrefix(line, instruction) {
				return line
			}
		}
	}
	return ""
}

// The mail settings may change only the last layer of the php image. Until
// 4.5.67 sendmail_path was written in the middle (php/ini), and a muted clone —
// which sends to its own envelope address — rebuilt the 14 layers after it,
// 6-12 s and ~2 GiB per copy (measured 2026-11-20). 4.5.68 moved the value into
// the environment, which cron jobs never see; now it is the last step of the
// image, and the compose file carries nothing about it.
func TestTheMailSettingsChangeOnlyTheLastLayerOfThePhpImage(t *testing.T) {
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
		"mail off":            `sendmail_path = "/usr/sbin/sendmail -t -i"`,
		"mailpit":             `sendmail_path = "/usr/bin/msmtp -t --port=1025 --host=host.docker.internal"`,
		"relay with a sender": `sendmail_path = "/usr/bin/msmtp -t --port=587 --host=relay.example.com --from=shop@example.com"`,
		"muted clone":         `sendmail_path = "/usr/bin/msmtp -t --port=1025 --host=host.docker.internal"`,
	}

	var first, firstName string
	for name, overrides := range configs {
		dockerfile, compose := renderSendmail(t, overrides)

		if !strings.Contains(dockerfile, want[name]) {
			t.Errorf("%s: php.Dockerfile lacks %s", name, want[name])
		}
		if layer := layerAfterSendmail(dockerfile); layer != "" {
			t.Errorf("%s: a layer follows the sendmail line, so every clone rebuilds it: %s", name, layer)
		}
		if strings.Contains(compose, "SENDMAIL") {
			t.Errorf("%s: the compose file carries a sendmail value; php.ini is the only place for it", name)
		}

		rest := withoutSendmail(dockerfile)
		if first == "" {
			first, firstName = rest, name
			continue
		}
		if rest != first {
			t.Errorf("php.Dockerfile for %q differs from %q beyond the sendmail line", name, firstName)
		}
	}
}
