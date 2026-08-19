//go:build e2e

package e2e

import (
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestServiceRestartTouchesOnlyTheServiceNamed is the whole reason the command
// exists, stated as a measurement: one container is younger afterwards and the
// others are exactly as old as they were.
//
// `restart` cannot be a step of a deploy recipe — it stops every container of
// the project, `deployer` among them, which is the process running the recipe.
// So restarting after a deploy stayed a second step for a person, and on
// 2026-08-19 three of four projects on one machine were serving code from a
// release older than the one `current` pointed at. Two of the three who forgot
// knew about the trap. The gap closes only if a recipe can restart its own
// application without touching the container it is running in.
//
// The same precision is what makes it usable on a machine where several
// projects share a database: a project-wide restart there takes the database
// container down with it, and every other application on the machine with it.
func TestServiceRestartTouchesOnlyTheServiceNamed(t *testing.T) {
	p := newProject(t, "e2esvcrestart")

	p.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2esvcrestart.test",
	)
	p.run(20*time.Minute, "start")

	dbBefore := startedAt(t, p, "db")
	nginxBefore := startedAt(t, p, "nginx")

	// A name this project does not have restarts nothing, and says what the
	// project does have — the only useful thing to put next to a refusal.
	out, err := p.tryRun(2*time.Minute, "service:restart", "nosuchservice")
	if err == nil {
		t.Errorf("service:restart accepted a service this project does not have:\n%s", out)
	}
	requireContains(t, out, "Nothing was restarted", "the refusal of an unknown service name")
	requireContains(t, out, "nginx", "the list of services offered after a refusal")

	if got := startedAt(t, p, "db"); got != dbBefore {
		t.Errorf("a refused name restarted db anyway: %s → %s", dbBefore, got)
	}

	// No name at all asks for one. Falling back to the whole project here would
	// be the blunt tool wearing the precise name.
	if out, err := p.tryRun(2*time.Minute, "service:restart"); err == nil {
		t.Errorf("service:restart with no service name was accepted:\n%s", out)
	}

	p.run(5*time.Minute, "service:restart", "db")

	dbAfter := startedAt(t, p, "db")
	nginxAfter := startedAt(t, p, "nginx")

	if dbAfter == dbBefore {
		t.Errorf("db was not restarted: still started at %s", dbAfter)
	}
	if nginxAfter != nginxBefore {
		t.Errorf("nginx was restarted too (%s → %s); the command is not precise, "+
			"which is the entire point of it", nginxBefore, nginxAfter)
	}
}

// startedAt reads when a service's container last started.
//
// Asked of docker rather than of madock, deliberately: `status` answers running
// or not, and everything here is running before and after — the claim under
// test is narrower than that, and it is which container is younger. This is a
// measurement of what madock did, not a way of driving it.
func startedAt(t *testing.T, p *project, service string) string {
	t.Helper()

	container := "madock_" + p.name + "-" + service + "-1"
	out, err := exec.Command("docker", "inspect", "--format", "{{.State.StartedAt}}", container).CombinedOutput()
	if err != nil {
		t.Fatalf("could not read the start time of %s: %v\n%s", container, err, out)
	}

	started := strings.TrimSpace(string(out))
	if started == "" {
		t.Fatalf("docker reported no start time for %s", container)
	}
	return started
}
