//go:build e2e

package e2e

import (
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

	p.run(5*time.Minute, "setup", "-y",
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
