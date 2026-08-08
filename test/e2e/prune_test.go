//go:build e2e

package e2e

import (
	"strings"
	"testing"
	"time"
)

// TestPruneKeepsTheDataUnlessAskedToDropIt pins the difference between two
// invocations of a destructive command that look almost identical.
//
//	madock prune                  containers go, the database stays
//	madock prune --with-volumes   the database goes too
//
// One flag between "I will start this again in a minute" and "the data is
// gone". Nothing tested either half, and the failure mode of getting it wrong
// is silent in the direction that matters: prune without the flag taking the
// volumes anyway is discovered when the project comes back empty.
//
// The same volume-name trick as the removal test proves it: a project rebuilt
// under its own name gets its own volume back, so yesterday's rows either are
// there or are not.
func TestPruneKeepsTheDataUnlessAskedToDropIt(t *testing.T) {
	p := newProject(t, "e2eprune")

	p.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2eprune.test",
	)
	p.run(20*time.Minute, "start")

	p.freshTable("survivor", "(note VARCHAR(32))")
	p.query("INSERT INTO survivor VALUES ('written-before-prune')")

	p.run(5*time.Minute, "prune")

	// The containers are gone, so this is a start that has to create them —
	// which is the case that used to report success and do nothing.
	p.run(20*time.Minute, "start")
	requireContains(t, p.run(3*time.Minute, "status"), "db running", "the database after a plain prune")

	rows := p.query("SELECT note FROM survivor")
	requireContains(t, rows, "written-before-prune", "the row a plain prune must not touch")

	// Now the destructive one.
	p.run(5*time.Minute, "prune", "--with-volumes")
	p.run(20*time.Minute, "start")

	tables := p.query("SHOW TABLES")
	if strings.Contains(tables, "survivor") {
		t.Errorf("prune --with-volumes left the database behind:\n%s", tables)
	}
}
