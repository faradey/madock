package restore

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/faradey/madock/v4/src/helper/snapshotengine"
)

// The refusal has to reach the caller. It used to return nothing, and the
// command went on to rebuild the project and print "Snapshot restored
// successfully" with exit status 0 after restoring nothing.
func TestARestoreRefusedOnADeployerProjectSaysSo(t *testing.T) {
	conf := map[string]string{"deploy/enabled": "true"}

	err := RestoreSnapshot("shop", conf, 1, []string{"snapshot-2026-10-15-12-00-00"}, t.TempDir())
	if !errors.Is(err, errDeployManaged) {
		t.Errorf("RestoreSnapshot on a deployer project = %v, want errDeployManaged", err)
	}
}

// A snapshot whose databases an engine holds, restored by a build without that
// engine, must stop before the project is taken down. Getting past the check
// would reach docker.Down — and with no engine to put the databases back, the
// project would come up on whatever its volumes held, called restored.
func TestASnapshotHeldByAMissingEngineIsRefusedBeforeAnythingStops(t *testing.T) {
	dir := t.TempDir()
	name := "snapshot-2026-10-18-12-00-00"
	if err := os.MkdirAll(filepath.Join(dir, name), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := snapshotengine.WriteMarker(filepath.Join(dir, name), snapshotengine.Marker{Engine: "nope", Ref: "pool/shop@x"}); err != nil {
		t.Fatal(err)
	}

	err := RestoreSnapshot("shop", map[string]string{}, 1, []string{name}, dir)
	if !errors.Is(err, snapshotengine.ErrUnknown) {
		t.Fatalf("RestoreSnapshot = %v, want ErrUnknown", err)
	}
	if !strings.Contains(err.Error(), `"nope"`) {
		t.Errorf("the refusal should name the engine: %v", err)
	}
}

func TestANameMeansExactlyThatNameAndTheNewestOfIt(t *testing.T) {
	names := []string{
		"snapshot-backup-2026-10-14-12-00-00",
		"snapshot-backup-old-2026-01-01-12-00-00",
		"snapshot-backup-2026-10-15-09-00-00",
		"snapshot-other-2026-12-31-00-00-00",
	}

	if got := MatchName(names, "backup"); got != "snapshot-backup-2026-10-15-09-00-00" {
		t.Errorf("MatchName(backup) = %q, want the newest snapshot named exactly backup", got)
	}
	if got := MatchName(names, "backup-old"); got != "snapshot-backup-old-2026-01-01-12-00-00" {
		t.Errorf("MatchName(backup-old) = %q", got)
	}
	if got := MatchName(names, "snapshot-other-2026-12-31-00-00-00"); got != "snapshot-other-2026-12-31-00-00-00" {
		t.Errorf("an exact directory name should win, got %q", got)
	}
	if got := MatchName(names, "missing"); got != "" {
		t.Errorf("MatchName(missing) = %q, want none", got)
	}
}
