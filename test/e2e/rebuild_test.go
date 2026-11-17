//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestRebuildKeepsTheData covers the difference between `rebuild` and
// `project:remove`, which is one boolean deep in the code and everything to the
// person running it.
//
// Both take the containers down. Removal passes `withVolumes: true` and the
// database goes with them; rebuild passes false and it must not. Nothing in the
// output of either command says which one happened — the only way to know is to
// look afterwards, which by then is too late.
//
// Rebuild is also the command people reach for when something is wrong, so it
// is run in exactly the state where losing the database hurts most.
func TestRebuildKeepsTheData(t *testing.T) {
	p := newProject(t, "e2erebuild")

	p.run(setupTimeout, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2erebuild.test",
	)
	p.run(20*time.Minute, "start")

	p.freshTable("survivor", "(note VARCHAR(32))")
	p.query("INSERT INTO survivor VALUES ('written-before-rebuild')")

	// Generous: rebuild removes the containers and builds the images again.
	p.run(25*time.Minute, "rebuild")

	// A rebuild that leaves the project down is its own kind of failure, and one
	// that looks like success in the log.
	status := p.run(3*time.Minute, "status")
	for _, service := range []string{"app", "db", "nginx"} {
		requireContains(t, status, service+" running", "after rebuild, "+service)
	}

	rows := p.query("SELECT note FROM survivor")
	requireContains(t, rows, "written-before-rebuild", "the row written before the rebuild")
}

// TestAFailedBuildLeavesTheProjectRunning is the order of a rebuild, asked of
// docker rather than of the code.
//
// A rebuild used to take the containers down and only then build the images,
// inside `up --build`, so the project was down for the whole build. On
// 2026-11-17 a live Laravel project stopped answering for about eleven minutes
// while its php image was rebuilt from a slow package mirror. The images are
// now built while the old containers still run, and that has a consequence a
// test can see without timing anything: a build that fails stops nothing.
//
// The build is broken through the project's own override, .madock/docker/,
// which is where a person would break it. Before the change the same rebuild
// had already run `docker compose down` when the build failed, and every
// service of the project was gone.
func TestAFailedBuildLeavesTheProjectRunning(t *testing.T) {
	p := newProject(t, "e2erebuildfail")

	p.run(setupTimeout, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2erebuildfail.test",
	)
	p.run(20*time.Minute, "start")

	started := map[string]string{}
	for _, service := range []string{"app", "db", "nginx"} {
		started[service] = startedAt(t, p, service)
	}

	override := filepath.Join(p.runDir, ".madock", "docker", "Dockerfile")
	if err := os.MkdirAll(filepath.Dir(override), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(override, []byte("FROM alpine:3\nRUN echo broken-on-purpose && exit 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := p.tryRun(10*time.Minute, "rebuild")
	if err == nil {
		t.Fatalf("a rebuild whose image cannot be built succeeded:\n%s", out)
	}
	requireContains(t, out, "nothing was stopped", "the refusal should say the project was left alone")

	// startedAt fails the test outright for a container that is gone, which is
	// what the old order left behind.
	for service, before := range started {
		if after := startedAt(t, p, service); after != before {
			t.Errorf("%s was restarted or recreated by a rebuild whose build failed (started %s, now %s)", service, before, after)
		}
	}
	status := p.run(3*time.Minute, "status")
	for _, service := range []string{"app", "db", "nginx"} {
		requireContains(t, status, service+" running", "after a failed rebuild, "+service)
	}

	// With the override gone the same rebuild goes through, so the failure above
	// was the build and not something else about this project.
	if err := os.Remove(override); err != nil {
		t.Fatal(err)
	}
	p.run(25*time.Minute, "rebuild")
}
