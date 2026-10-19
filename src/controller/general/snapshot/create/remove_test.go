package create

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/faradey/madock/v4/src/helper/snapshotengine"
)

// pinEngine refuses to drop the snapshots in pinned — what zfs does while a
// clone still lives on one — and records what it was asked to drop.
type pinEngine struct {
	pinned  map[string]bool
	dropped *[]string
}

func (pinEngine) Live() bool                                                 { return true }
func (pinEngine) Capture(snapshotengine.Project, string) (string, error)     { return "", nil }
func (pinEngine) Restore(snapshotengine.Project, string) error               { return nil }
func (pinEngine) Clone(snapshotengine.Project, snapshotengine.Project) error { return nil }
func (e pinEngine) Drop(_ snapshotengine.Project, ref string) error {
	if e.pinned[ref] {
		return errors.New("snapshot has dependent clones")
	}
	*e.dropped = append(*e.dropped, ref)
	return nil
}

var dropped []string

func init() {
	snapshotengine.Register("prune-test", pinEngine{pinned: map[string]bool{"ref-oldest": true}, dropped: &dropped})
}

func snapshotDir(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "projects", "shop", "backup", "snapshot")
}

func makeSnapshot(t *testing.T, dir, name, engineRef string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if engineRef == "" {
		if err := os.WriteFile(filepath.Join(path, "db.tar.gz"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	if err := snapshotengine.WriteMarker(path, snapshotengine.Marker{Engine: "prune-test", Ref: engineRef}); err != nil {
		t.Fatal(err)
	}
}

func TestKeepWithoutAValueIsNoLimit(t *testing.T) {
	for _, v := range []string{"", "0", "-3", "many"} {
		if Keep(map[string]string{"snapshot/keep": v}) != 0 {
			t.Errorf("snapshot/keep %q should mean no limit", v)
		}
	}
	dir := snapshotDir(t)
	makeSnapshot(t, dir, "snapshot-2026-01-01-00-00-00", "")
	if removed := Prune(dir, 0); len(removed) != 0 {
		t.Errorf("no limit removed %v", removed)
	}
}

// The oldest goes first, by the timestamp and not by the name: "zzz" made in
// January is older than "aaa" made in March.
func TestPruneRemovesTheOldestByTimestamp(t *testing.T) {
	dir := snapshotDir(t)
	makeSnapshot(t, dir, "snapshot-zzz-2026-01-01-00-00-00", "")
	makeSnapshot(t, dir, "snapshot-aaa-2026-03-01-00-00-00", "")
	makeSnapshot(t, dir, "snapshot-2026-02-01-00-00-00", "")

	removed := Prune(dir, 2)
	if strings.Join(removed, ",") != "snapshot-zzz-2026-01-01-00-00-00" {
		t.Fatalf("removed %v, want the January one", removed)
	}
	if got := strings.Join(OrderedNames(dir), ","); got != "snapshot-2026-02-01-00-00-00,snapshot-aaa-2026-03-01-00-00-00" {
		t.Fatalf("left %s", got)
	}
}

// A snapshot a clone still lives on cannot be dropped. It stays — directory
// included, so it is still listed and restorable — and the next oldest goes in
// its place, so one pinned snapshot does not stop the cleaning.
func TestAPinnedSnapshotStaysAndTheNextOldestGoes(t *testing.T) {
	dropped = nil
	dir := snapshotDir(t)
	makeSnapshot(t, dir, "snapshot-2026-01-01-00-00-00", "ref-oldest")
	makeSnapshot(t, dir, "snapshot-2026-02-01-00-00-00", "ref-middle")
	makeSnapshot(t, dir, "snapshot-2026-03-01-00-00-00", "ref-newest")

	removed := Prune(dir, 2)
	if strings.Join(removed, ",") != "snapshot-2026-02-01-00-00-00" {
		t.Fatalf("removed %v, want the middle one", removed)
	}
	if strings.Join(dropped, ",") != "ref-middle" {
		t.Fatalf("the engine dropped %v, want ref-middle", dropped)
	}
	if _, err := os.Stat(filepath.Join(dir, "snapshot-2026-01-01-00-00-00", snapshotengine.MarkerFile)); err != nil {
		t.Fatalf("the pinned snapshot's directory went with the refusal: %v", err)
	}
}

// The engine's snapshot is dropped before the directory goes. A directory
// removed while its engine snapshot stays is space nothing will ever list.
func TestRemoveKeepsTheDirectoryWhenTheEngineIsMissing(t *testing.T) {
	dir := snapshotDir(t)
	path := filepath.Join(dir, "snapshot-2026-01-01-00-00-00")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := snapshotengine.WriteMarker(path, snapshotengine.Marker{Engine: "no-such-engine", Ref: "r"}); err != nil {
		t.Fatal(err)
	}
	if err := Remove(dir, "snapshot-2026-01-01-00-00-00"); !errors.Is(err, snapshotengine.ErrUnknown) {
		t.Fatalf("removing a snapshot held by a missing engine: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("the directory was removed although its engine snapshot could not be: %v", err)
	}
}
