//go:build e2e

package e2e

import (
	"strings"
	"testing"
	"time"
)

// TestRestartReadsArgumentsBeforeStopping pins the one rule that turns a typo
// back into a typo: argument parsing has to finish before the first side
// effect.
//
// `restart` was stop-then-start with no parsing of its own, and the parsing
// lived inside `start` — after everything was already down. Anything the
// command does not understand therefore reached the parser with the environment
// stopped, and the parser ends the process. Two measured cases, both on
// production machines on 2026-08-18:
//
//   - `madock restart php`, meant as "restart just this service", stopped
//     nginx, php, db, redisdb and deployer and left them stopped. The message,
//     "too many positional arguments at 'php'", reads as though nothing
//     happened;
//   - `madock restart --help` did the same to a shared-database provider,
//     taking the cluster's database down with it — and `--help` is typed
//     exactly when somebody is unsure of the syntax. Asking how to use a
//     command must not be how you find out.
//
// The assertion is the same for both: the containers are still running
// afterwards.
func TestRestartReadsArgumentsBeforeStopping(t *testing.T) {
	p := newProject(t, "e2erestart")

	p.run(3*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2erestart.test",
	)
	p.run(20*time.Minute, "start")

	running := func(step string) {
		t.Helper()
		status := p.run(1*time.Minute, "status")
		if !strings.Contains(status, "running") {
			t.Fatalf("after %s nothing is running:\n%s", step, status)
		}
	}

	running("start")

	// An argument the command does not take. It has to fail — and leave the
	// project exactly as it was.
	if out, err := p.tryRun(2*time.Minute, "restart", "php"); err == nil {
		t.Errorf("`restart php` was accepted; it takes no positional argument:\n%s", out)
	}
	running("`restart php`")

	// --help is answered, not acted on.
	help := p.run(2*time.Minute, "restart", "--help")
	if !strings.Contains(help, "with-chown") {
		t.Errorf("`restart --help` did not describe the command's arguments:\n%s", help)
	}
	running("`restart --help`")

	// And the command itself still works.
	p.run(10*time.Minute, "restart")
	running("`restart`")
}
