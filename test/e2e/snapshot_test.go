//go:build e2e

package e2e

import (
	"strings"
	"testing"
	"time"
)

// TestSnapshotRestoresTheDatabase is the round trip: write something, take a
// snapshot, change it, restore, and find the original back.
//
// Snapshots are the feature whose failure is silent. Nothing about a bad
// snapshot is visible on the day it is taken — it is discovered by the person
// who needed it, at the moment they needed it, which is the worst possible
// audience for the news. Nothing else in madock has that property.
//
// The database is what the test writes to, deliberately. It is the part of a
// snapshot that is hardest to get right: a data directory copied out of a
// running server comes out torn, which is why `snapshot:create` stops the
// project first.
func TestSnapshotRestoresTheDatabase(t *testing.T) {
	p := newProject(t, "e2esnap")

	p.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2esnap.test",
	)
	p.run(20*time.Minute, "start")

	p.freshTable("probe", "(note VARCHAR(32))")
	p.query("INSERT INTO probe VALUES ('before-snapshot')")

	p.run(10*time.Minute, "snapshot:create", "-n", "probe-point")

	// The project is running again by now — snapshot:create stops it, copies,
	// and starts it back up. If that were not so, this write would fail and say
	// so plainly.
	p.query("INSERT INTO probe VALUES ('after-snapshot')")

	both := p.query("SELECT note FROM probe")
	requireContains(t, both, "before-snapshot", "the row written before the snapshot")
	requireContains(t, both, "after-snapshot", "the row written after it")

	// A name that does not exist must stop rather than fall back to the prompt,
	// which would hang a script, or pick something else, which would restore the
	// wrong data.
	out, err := p.tryRun(2*time.Minute, "snapshot:restore", "--name", "no-such-snapshot")
	if err == nil {
		t.Errorf("restoring a snapshot that does not exist succeeded:\n%s", out)
	}
	requireContains(t, out, "probe-point", "the error should list the snapshots that do exist")

	// Restoring by name. It ends with a rebuild, so it takes a while.
	p.run(20*time.Minute, "snapshot:restore", "--name", "probe-point")

	restored := p.query("SELECT note FROM probe")
	requireContains(t, restored, "before-snapshot", "the row the snapshot contained")
	if strings.Contains(restored, "after-snapshot") {
		t.Errorf("the row written after the snapshot survived the restore:\n%s", restored)
	}

	// The other branch: no name, a numbered list, a choice on stdin. This is
	// what a person does, and it is the older of the two paths — worth keeping
	// covered now that a second one exists beside it.
	p.query("INSERT INTO probe VALUES ('after-restore')")
	p.runWithInput(20*time.Minute, "1\n", "snapshot:restore")

	again := p.query("SELECT note FROM probe")
	requireContains(t, again, "before-snapshot", "the row the snapshot contained")
	if strings.Contains(again, "after-restore") {
		t.Errorf("the interactive restore did not roll the database back:\n%s", again)
	}
}
