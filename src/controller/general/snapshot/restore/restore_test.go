package restore

import (
	"errors"
	"testing"
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
