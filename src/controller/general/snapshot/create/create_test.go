package create

import (
	"errors"
	"os/exec"
	"testing"
)

// exitWith runs a shell that exits with the given status and returns the error,
// so the test sees a real *exec.ExitError rather than a hand-built stand-in.
func exitWith(t *testing.T, status string) error {
	t.Helper()
	err := exec.Command("sh", "-c", "exit "+status).Run()
	if err == nil {
		t.Fatalf("sh -c 'exit %s' reported success", status)
	}
	return err
}

// tar exits 1 when a member changed while it was being read. The archive is
// complete, so a snapshot of the project files survives it — that is the normal
// state of a project whose containers are up, and failing there is what made
// snapshot:create unusable on a running environment.
func TestIsFilesChangedOnStatusOne(t *testing.T) {
	if !isFilesChanged(exitWith(t, "1")) {
		t.Error("exit status 1 not recognised as a changed file")
	}
}

// Everything above 1 is a fatal tar error — an unreadable member, a full disk,
// a broken pipe. Swallowing those would store an archive that cannot be
// restored and call it a snapshot.
func TestIsFilesChangedRejectsFatalStatuses(t *testing.T) {
	for _, status := range []string{"2", "3", "127"} {
		if isFilesChanged(exitWith(t, status)) {
			t.Errorf("exit status %s treated as a changed file", status)
		}
	}
}

func TestIsFilesChangedIgnoresOtherErrors(t *testing.T) {
	if isFilesChanged(nil) {
		t.Error("a successful run reported a changed file")
	}
	if isFilesChanged(errors.New("docker: command not found")) {
		t.Error("a non-exit error reported a changed file")
	}
}
