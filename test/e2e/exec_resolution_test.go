//go:build e2e

package e2e

import (
	"strings"
	"testing"
	"time"
)

// TestExecResolvesContainerAndUser covers the rules every exec-shaped command
// shares, once, rather than covering the commands one at a time.
//
// `bash`, `cli`, `composer`, `node`, `n98` and `magento` differ in what they
// run and agree on how they decide where to run it: a service (the platform's
// main container, or a hardcoded one, or the `-s` flag), a user (a default the
// command picks, or `-u`), and the environment overriding both through one
// helper — `cli.GetEnvForUserServiceWorkdir`. Every one of those commands has
// been wrong about at least one of the three at least once.
//
// Only `cli` had a test, and only for the user half. This adds the other two
// rules where they can be observed cheaply, on a project with no PHP at all.
func TestExecResolvesContainerAndUser(t *testing.T) {
	p := newProject(t, "e2eexecres")

	p.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2eexecres.test",
	)
	p.run(20*time.Minute, "start")

	// The main service for language=none is `app`, and `cli` is the command
	// that takes it without being told. Its hostname is the container id, which
	// is what makes the later comparisons mean something.
	appHost := containerHostname(t, p.run(2*time.Minute, "cli", "hostname"))
	if appHost == "" {
		t.Fatalf("cli hostname printed nothing usable")
	}

	// -s picks a different service. bash is interactive, but with no TTY madock
	// execs with -i alone, so a command on stdin is how it is driven.
	//
	// The hostname says it is a different container; /var/lib/mysql says which
	// one. Without the second half the test would pass against a bash that
	// landed in any container that was not `app`.
	dbShell := p.runWithInput(2*time.Minute, "hostname\nls -d /var/lib/mysql\n", "bash", "-s", "db")
	dbHost := containerHostname(t, dbShell)
	if dbHost == appHost {
		t.Errorf("bash -s db ran in the same container as cli (%s):\n%s", appHost, dbShell)
	}
	requireContains(t, dbShell, "/var/lib/mysql", "bash -s db landing in the database container")

	// -u picks the user. bash defaults to root, so asking for a non-root user
	// is the only way round that can fail.
	asWWW := p.runWithInput(2*time.Minute, "id -un\n", "bash", "-s", "app", "-u", "www-data")
	requireContains(t, asWWW, "www-data", "bash -u www-data")

	// MADOCK_USER is how a command is run as root without a flag — it is what
	// the documentation tells people to use, and nothing has ever checked that
	// it still reaches the exec.
	asRoot, err := p.tryRunWith(2*time.Minute, "", []string{"MADOCK_USER=root"}, "cli", "id", "-un")
	if err != nil {
		t.Fatalf("cli under MADOCK_USER=root failed: %v\n%s", err, asRoot)
	}
	if !strings.Contains(asRoot, "root") {
		t.Errorf("MADOCK_USER=root did not reach the container:\n%s", asRoot)
	}

	// MADOCK_SERVICE_NAME is the same override for the container. Redirecting
	// `cli`, whose service is otherwise decided by the platform, is the case
	// that proves the environment wins over that decision.
	redirected, err := p.tryRunWith(2*time.Minute, "", []string{"MADOCK_SERVICE_NAME=db"}, "cli", "hostname")
	if err != nil {
		t.Fatalf("cli under MADOCK_SERVICE_NAME=db failed: %v\n%s", err, redirected)
	}
	if got := containerHostname(t, redirected); got != dbHost {
		t.Errorf("MADOCK_SERVICE_NAME=db sent cli to %q, but the database container is %q:\n%s",
			got, dbHost, redirected)
	}
}

// TestExecCommandsFailLoudlyWithNoContainer is the other half of the resolution
// rules: what happens when the service a command hardcodes does not exist here.
//
// `composer` always resolves to `php` and `node` always to `nodejs`. On a
// project with neither — every `custom` project, and every Node platform for
// composer — the container is simply not there. That is fine; the requirement
// is that the command says so and exits non-zero, because these are what CI
// scripts and our own release tooling call. A composer that cannot run and
// exits 0 is a build that reports success without installing anything.
func TestExecCommandsFailLoudlyWithNoContainer(t *testing.T) {
	p := newProject(t, "e2eexecmiss")

	p.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2eexecmiss.test",
	)
	p.run(20*time.Minute, "start")

	for _, c := range []struct {
		command   string
		container string
		args      []string
	}{
		{"composer", "e2eexecmiss-php-1", []string{"composer", "--version"}},
		{"node", "e2eexecmiss-nodejs-1", []string{"node", "node --version"}},
	} {
		out, err := p.tryRun(2*time.Minute, c.args...)
		if err == nil {
			t.Errorf("%s exited 0 with no container to run in:\n%s", c.command, out)
		}
		// Naming the container it looked for is what separates "the service is
		// not here" from a command that fell over parsing its own arguments —
		// both exit non-zero, and only one of them is this test's subject.
		requireContains(t, out, c.container, c.command+" reporting which container it wanted")
	}
}

// containerHostname returns the last non-empty line that looks like a hostname.
//
// madock may print lines of its own around the command's output, and the
// container id is what the shell prints last for `hostname`.
func containerHostname(t *testing.T, out string) string {
	t.Helper()

	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(line, " ") || strings.Contains(line, "/") {
			continue
		}
		return line
	}
	return ""
}
