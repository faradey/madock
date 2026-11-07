//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestStartExitStatusFollowsWhatItReports starts a project with a service that
// finished and then with one that died. The first is what a one-shot does and
// `start` succeeds; the second means the project is not up, and a script that
// has only the exit status has to be able to tell.
func TestStartExitStatusFollowsWhatItReports(t *testing.T) {
	p := newProject(t, "e2estatus")
	p.run(setupTimeout, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2estatus.test",
	)

	override := filepath.Join(p.runDir, ".madock", "docker", "docker-compose.linux.yml")
	if err := os.MkdirAll(filepath.Dir(override), 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(body string) {
		t.Helper()
		if err := os.WriteFile(override, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write(`services:
  finished:
    image: alpine:3
    entrypoint: ["/bin/sh", "-c", "exit 0"]
`)
	out := p.run(20*time.Minute, "start")
	requireContains(t, out, "finished", "a finished service is still reported")

	write(`services:
  doomed:
    image: alpine:3
    entrypoint: ["/bin/sh", "-c", "exit 3"]
`)
	out, err := p.tryRun(5*time.Minute, "start")
	if err == nil {
		t.Fatalf("start exited 0 with a service dead:\n%s", out)
	}
	requireContains(t, out, "doomed", "the dead service is named")

	// `start` names it once. Afterwards plain `status` lists a stopped service only
	// when it is a worker, so --all is what answers without reaching for docker.
	out = p.run(2*time.Minute, "status")
	if strings.Contains(out, "doomed") {
		t.Errorf("plain status lists a stopped service that is not a worker:\n%s", out)
	}
	out = p.run(2*time.Minute, "status", "--all")
	requireContains(t, out, "doomed exited (exit 3)", "status --all names the dead service and how it ended")
	out = p.run(2*time.Minute, "status", "--all", "--json")
	requireContains(t, out, `"exit_code": 3`, "and so does its JSON")
}
