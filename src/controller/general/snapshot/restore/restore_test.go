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
