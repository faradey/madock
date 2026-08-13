//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestCommandsRefuseBeforeTouchingDocker covers `diff`, `patch:create` and
// `setup:env` at the only place they can be covered without a Magento project:
// the moment they decide not to run.
//
// All three end in `docker exec … php /var/www/scripts/php/…`, so their happy
// paths need a php container and a real store — which is the platform test
// further down the queue, not this one. What can be pinned here is the half
// that runs first and matters most: a command missing an argument has to stop
// while it is still holding it, rather than build a command line around an
// empty string and hand it to a container.
//
// The assertion is not the exit code alone. Reaching docker is visible from
// outside — a `custom` project has no php container, so anything that got that
// far says "No such container". Its absence is what proves the refusal happened
// earlier.
//
// No `start` here on purpose: nothing under test is supposed to need a running
// project, and a test that starts one would be paying twenty seconds to hide
// exactly the mistake it is looking for.
func TestCommandsRefuseBeforeTouchingDocker(t *testing.T) {
	p := newProject(t, "e2eguards")

	p.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2eguards.test",
	)

	for _, c := range []struct {
		what string
		args []string
		says string
	}{
		{
			what: "diff with no --platform",
			args: []string{"diff", "--old=2.4.6", "--new=2.4.7"},
			says: "platform",
		},
		{
			what: "diff with no --old or --new",
			args: []string{"diff", "--platform=magento"},
			says: "old",
		},
		{
			what: "diff for a platform it does not know",
			args: []string{"diff", "--platform=shopware", "--old=6.6", "--new=6.7"},
			says: "shopware",
		},
		{
			what: "patch:create with no --file",
			args: []string{"patch:create", "-n", "somepatch"},
			says: "file",
		},
	} {
		out, err := p.tryRun(time.Minute, c.args...)
		if err == nil {
			t.Errorf("%s exited 0:\n%s", c.what, out)
		}
		if strings.Contains(out, "No such container") {
			t.Errorf("%s reached docker before refusing:\n%s", c.what, out)
		}
		if !strings.Contains(strings.ToLower(out), c.says) {
			t.Errorf("%s did not say which argument was missing (expected %q):\n%s", c.what, c.says, out)
		}
	}

	// setup:env is the one with something to destroy. env.php holds the
	// credentials of whatever the project is connected to, and this command
	// writes it — so the refusal when one already exists is the only thing
	// standing between a stray `setup:env` and a store that cannot find its
	// database.
	envDir := filepath.Join(p.runDir, "app", "etc")
	if err := os.MkdirAll(envDir, 0o755); err != nil {
		t.Fatalf("creating %s: %v", envDir, err)
	}
	envFile := filepath.Join(envDir, "env.php")
	original := "<?php return ['must' => 'survive'];\n"
	if err := os.WriteFile(envFile, []byte(original), 0o644); err != nil {
		t.Fatalf("writing %s: %v", envFile, err)
	}

	out, err := p.tryRun(time.Minute, "setup:env")
	if err == nil {
		t.Errorf("setup:env overwrote an existing env.php without being forced:\n%s", out)
	}
	if strings.Contains(out, "No such container") {
		t.Errorf("setup:env reached docker before noticing the existing env.php:\n%s", out)
	}

	after, readErr := os.ReadFile(envFile)
	if readErr != nil {
		t.Fatalf("env.php is gone after a refused setup:env: %v", readErr)
	}
	if string(after) != original {
		t.Errorf("a refused setup:env changed env.php:\n%s", after)
	}
}
