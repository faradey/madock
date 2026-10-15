package list

import (
	"os"
	"path/filepath"
	"testing"
)

// A name given at creation sorts as text, so "zzz" made first would be listed
// after "aaa" made later. Oldest first means by the timestamp.
func TestSnapshotsAreListedOldestFirstWhateverTheirName(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{
		"snapshot-aaa-2026-10-15-12-00-00",
		"snapshot-zzz-2026-10-14-12-00-00",
		"snapshot-2026-10-13-12-00-00",
	} {
		if err := os.MkdirAll(filepath.Join(dir, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "snapshot-aaa-2026-10-15-12-00-00", "db.tar.gz"), make([]byte, 7), 0o644); err != nil {
		t.Fatal(err)
	}

	got := Read(dir)
	want := []string{"snapshot-2026-10-13-12-00-00", "snapshot-zzz-2026-10-14-12-00-00", "snapshot-aaa-2026-10-15-12-00-00"}
	if len(got) != len(want) {
		t.Fatalf("got %d snapshots, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].Name != want[i] {
			t.Errorf("position %d: %s, want %s", i, got[i].Name, want[i])
		}
	}
	if last := got[2]; last.SizeBytes != 7 || len(last.Files) != 1 || last.Files[0] != "db.tar.gz" {
		t.Errorf("unexpected contents: %+v", last)
	}
}

func TestNoSnapshotDirectoryIsAnEmptyList(t *testing.T) {
	got := Read(filepath.Join(t.TempDir(), "missing"))
	if got == nil || len(got) != 0 {
		t.Errorf("expected an empty, non-nil list, got %#v", got)
	}
}
