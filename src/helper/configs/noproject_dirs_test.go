package configs

import (
	"os"
	"path/filepath"
	"testing"
)

// Asking from a directory that is no project leaves nothing in the
// installation. It created projects/<dir>/docker/{nginx,php} for every such
// directory, so a `config:set --global` from a home directory registered an
// empty project shell named after it (Indemorio IND-14).
func TestAQuestionOutsideAProjectCreatesNothing(t *testing.T) {
	tmpExec := t.TempDir()
	t.Setenv("MADOCK_EXEC_DIR", tmpExec)
	t.Setenv("MADOCK_RUN_DIR", t.TempDir())
	CleanCache()
	t.Cleanup(CleanCache)

	if IsHasConfig("STRANGER") {
		t.Fatal("a directory with no config was taken for a project")
	}
	if _, err := os.Stat(filepath.Join(tmpExec, "projects", "STRANGER")); !os.IsNotExist(err) {
		t.Errorf("projects/STRANGER was created for a directory that is no project (%v)", err)
	}
}
