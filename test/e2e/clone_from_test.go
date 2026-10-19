//go:build e2e

package e2e

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// TestCloneFromASnapshotTakesItsSavedState — `project:clone` copies the source
// as it is at that moment, stopping it to do so. `--from` is the other half: a
// clone of a saved state, for the copy that has to start from the same point
// every time (a demo, a reproduction), and it must not touch the source at all.
//
// snapshot/keep rides along because it decides which saved states still exist
// to clone from.
func TestCloneFromASnapshotTakesItsSavedState(t *testing.T) {
	install := newInstallation(t)
	source := install.project("e2efrom")

	source.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2efrom.test",
	)
	source.run(2*time.Minute, "config:set", "-n", "snapshot/keep", "-v", "2")
	source.run(20*time.Minute, "start")

	source.freshTable("state", "(note VARCHAR(32))")
	source.query("INSERT INTO state VALUES ('saved')")
	source.run(10*time.Minute, "snapshot:create", "-n", "golden")
	source.query("INSERT INTO state VALUES ('written-after')")

	// A name that is not there is refused before a project is created.
	if out, err := source.tryRun(2*time.Minute, "project:clone", "-n", "e2efromnone", "-s=-none", "--from", "no-such"); err == nil {
		t.Errorf("--from a missing snapshot succeeded:\n%s", out)
	}

	source.run(30*time.Minute, "project:clone", "-n", "e2efromcopy", "-s=-copy", "--from", "golden")
	clone := install.project("e2efromcopy")
	clone.run(20*time.Minute, "start")

	got := clone.query("SELECT note FROM state")
	requireContains(t, got, "saved", "the clone should hold the snapshot's row")
	if strings.Contains(got, "written-after") {
		t.Errorf("the clone holds a row written after the snapshot — it copied the source, not the snapshot:\n%s", got)
	}
	// The source kept running and kept its own state.
	requireContains(t, source.query("SELECT note FROM state"), "written-after", "the source's own row")

	// keep=2: a third snapshot removes the oldest, which is "golden".
	source.run(10*time.Minute, "snapshot:create", "-n", "second")
	source.run(10*time.Minute, "snapshot:create", "-n", "third")
	var listed struct {
		Data struct {
			Snapshots []struct {
				Name string `json:"name"`
			} `json:"snapshots"`
		} `json:"data"`
	}
	raw := source.run(2*time.Minute, "snapshot:list", "--json")
	if err := json.Unmarshal([]byte(raw[strings.Index(raw, "{"):]), &listed); err != nil {
		t.Fatalf("snapshot:list --json: %v\n%s", err, raw)
	}
	var names []string
	for _, s := range listed.Data.Snapshots {
		names = append(names, s.Name)
	}
	if len(names) != 2 || strings.Contains(strings.Join(names, ","), "golden") {
		t.Errorf("with snapshot/keep 2 the snapshots are %v, want the two newest", names)
	}

	source.run(2*time.Minute, "snapshot:remove", "-n", "second")
	raw = source.run(2*time.Minute, "snapshot:list")
	if strings.Contains(raw, "second") {
		t.Errorf("snapshot:remove left the snapshot listed:\n%s", raw)
	}
}
