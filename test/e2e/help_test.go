//go:build e2e

package e2e

import (
	"strings"
	"testing"
	"time"
)

// TestHelpIsAnsweredNotActedOn runs `--help` against the commands that used to
// run instead of answering it.
//
// Answering `--help` used to be each command's own job, done as a side effect
// of calling the argument parser. A command with no arguments to parse never
// called one — and so did what it does. `madock install --help` on an installed
// project printed the assembled `bin/magento setup:install --base-url=…
// --admin-password=…` line, password included, and ran it over the live
// database, reaching "Enabling Maintenance Mode" before it stopped on the
// existing env.php. Found on a live stand, 2026-08-20.
//
// Eight commands in madock were in that state and over fifty in madock-pro, so
// the check lives in the dispatcher now and this test reads it there: the
// assertion is the same sentence for every command, which is the point of
// having moved it.
//
// `restart --help` is covered separately in restart_test.go and is not repeated
// here. Worth knowing why that one existed and this one did not: `restart`
// parsed its arguments before stopping anything, so it was already answering
// --help correctly — the e2e suite covered the one command that was never
// broken, and none of the eight that were.
func TestHelpIsAnsweredNotActedOn(t *testing.T) {
	p := newProject(t, "e2ehelp")

	p.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2ehelp.test",
	)

	// No `start` for this half on purpose: none of these needs a running
	// project to prove the point, and the ones that would have acted leave
	// evidence either way — `install` on a custom platform answers "This
	// command is not supported for custom", `compress` writes an archive, and
	// `ssl:rebuild`, measured against a pre-fix binary on 2026-08-20, generated
	// a key pair and asked for the password to add a certificate to the system
	// trust store. Any of those in place of the help block is the failure.
	//
	// `stop` is in this list but the text alone does not cover it, which is
	// worth knowing before trusting a green run here: against the same pre-fix
	// binary `madock stop --help` took ten seconds, stopped the project and
	// *then* printed the help — something further down its path parses
	// arguments, so the help arrives after the work. Only state proves that
	// one, which is what the next test is for.
	for _, name := range []string{
		"install",
		"stop",
		"ssl:rebuild",
		"mftf:init",
		"compress",
		"uncompress",
		"config:cache:clean",
		"mcp",
	} {
		t.Run(name, func(t *testing.T) {
			out := p.run(2*time.Minute, name, "--help")
			requireContains(t, out, "Command: "+name, "`madock "+name+" --help`")
		})
	}

	// A pass-through command is the other half of the rule, and it has to keep
	// working: `composer --help` is composer's help to answer, not madock's, so
	// the dispatcher must not intercept it. On a custom project with no php
	// container the command gets as far as docker and fails there — which is
	// itself the proof that madock did not answer it.
	out, _ := p.tryRun(2*time.Minute, "composer", "--help")
	if strings.Contains(out, "Command: composer") {
		t.Errorf("madock answered `composer --help` itself; the arguments are composer's:\n%s", out)
	}
}

// TestStopHelpLeavesTheProjectRunning is the same bug where it costs something.
//
// The sweep above proves the help text appears. This proves the command did not
// happen — and `stop` is the one where that is measurable without a Magento
// install: before the dispatcher check, `madock stop --help` stopped every
// container in the project. On a shared-database provider that is the database
// every other application on the machine is connected to, and `--help` is typed
// exactly when somebody is unsure of a command. Asking how to use one must not
// be how you find out.
func TestStopHelpLeavesTheProjectRunning(t *testing.T) {
	p := newProject(t, "e2ehelpstop")

	p.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2ehelpstop.test",
	)
	p.run(20*time.Minute, "start")

	services := []string{"app", "db", "nginx"}

	// Asked as JSON and only about the project's own services: the text form
	// has a Proxy section whose container is also called nginx, so matching the
	// whole output would pass or fail on the state of a shared proxy.
	projectServices := func(what string) map[string]bool {
		t.Helper()

		var payload struct {
			Data struct {
				Services []struct {
					Service string `json:"service"`
					Running bool   `json:"running"`
				} `json:"services"`
			} `json:"data"`
		}
		out := p.run(3*time.Minute, "status", "--json")
		decode(t, out, &payload, "status --json "+what)

		running := map[string]bool{}
		for _, service := range payload.Data.Services {
			running[service.Service] = service.Running
		}
		return running
	}

	// Named services, not a walk of whatever came back: an empty list would
	// satisfy a range loop without executing it once, and this test would then
	// pass by asking nothing — which is the shape of the bug it exists for.
	before := projectServices("after start")
	for _, service := range services {
		if !before[service] {
			t.Fatalf("the project did not come up, so this test cannot say anything: %s is not running: %v", service, before)
		}
	}

	help := p.run(2*time.Minute, "stop", "--help")
	requireContains(t, help, "Command: stop", "`madock stop --help`")

	after := projectServices("after `stop --help`")
	for _, service := range services {
		if !after[service] {
			t.Errorf("`stop --help` stopped %s; asking what a command does must not do it: %v", service, after)
		}
	}

	// And the command itself still works — a help check that answered
	// everything would be its own kind of bug.
	p.run(5*time.Minute, "stop")

	stopped := projectServices("after stop")
	for _, service := range services {
		if stopped[service] {
			t.Errorf("after stop, %s is still running: %v", service, stopped)
		}
	}
}
