package create

import (
	"testing"

	"github.com/faradey/madock/v4/src/helper/snapshotengine"
)

// sizedEngine reports a size for its snapshots, as the zfs engine does.
type sizedEngine struct{ pinEngine }

func (sizedEngine) SnapshotBytes(_ snapshotengine.Project, ref string) (int64, error) {
	if ref == "ref-sized" {
		return 9_700_000_000, nil
	}
	return 0, nil
}

func init() {
	snapshotengine.Register("sized-test", sizedEngine{pinEngine{pinned: map[string]bool{}, dropped: &dropped}})
}

// A snapshot an engine took leaves engine.json alone in its directory, so
// size_bytes is a few dozen bytes for gigabytes of database (Indemorio
// IND-15). The engine's own figure comes back as engine_bytes; an engine that
// cannot say gives none, rather than a zero that reads as an empty snapshot.
func TestAnEngineSnapshotReportsTheEnginesSize(t *testing.T) {
	dir := snapshotDir(t)
	makeSnapshot(t, dir, "snapshot-2026-01-01-00-00-00", "")
	if err := snapshotengine.WriteMarker(dir+"/snapshot-2026-01-01-00-00-00", snapshotengine.Marker{Engine: "sized-test", Ref: "ref-sized"}); err != nil {
		t.Fatal(err)
	}
	got := DescribeFor(shop, dir, "snapshot-2026-01-01-00-00-00")
	if got.EngineBytes == nil || *got.EngineBytes != 9_700_000_000 {
		t.Fatalf("engine_bytes = %v", got.EngineBytes)
	}

	makeSnapshot(t, dir, "snapshot-2026-02-01-00-00-00", "ref-middle") // prune-test: no Sizer
	if got := DescribeFor(shop, dir, "snapshot-2026-02-01-00-00-00"); got.EngineBytes != nil {
		t.Errorf("an engine that cannot say reported %d", *got.EngineBytes)
	}
}
