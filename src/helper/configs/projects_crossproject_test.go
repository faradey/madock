package configs

import (
	"os"
	"path/filepath"
	"testing"
)

// TestGetProjectConfigOnly_CrossProjectDoesNotStealCWD verifies that resolving a
// foreign project's config (one whose runtime config has no `path`) does NOT fall
// back to the caller's CWD and merge the caller's release .madock/config.xml.
func TestGetProjectConfigOnly_CrossProjectDoesNotStealCWD(t *testing.T) {
	tmpExec := t.TempDir()
	tmpRun := t.TempDir() // pretend this is the *current* project's source dir
	t.Setenv("MADOCK_EXEC_DIR", tmpExec)
	t.Setenv("MADOCK_RUN_DIR", tmpRun)

	// Release config of the current project (CWD). If the fallback leaked, this
	// sentinel would show up when resolving project "B".
	if err := os.MkdirAll(filepath.Join(tmpRun, ".madock"), 0o755); err != nil {
		t.Fatal(err)
	}
	SaveInFile(filepath.Join(tmpRun, ".madock", "config.xml"), map[string]string{"ssh/host": "FROM_CWD"}, "default")

	// Runtime config of foreign project "B" WITHOUT a `path` key.
	if err := os.MkdirAll(filepath.Join(tmpExec, "projects", "B"), 0o755); err != nil {
		t.Fatal(err)
	}
	SaveInFile(filepath.Join(tmpExec, "projects", "B", "config.xml"), map[string]string{"db/database": "B_RUNTIME"}, "default")

	CleanCache()
	SetProjectNameResolver(func() string { return "A" })
	t.Cleanup(func() { SetProjectNameResolver(nil); CleanCache() })

	conf := GetProjectConfigOnly("B")

	if got := conf["db/database"]; got != "B_RUNTIME" {
		t.Errorf("db/database = %q, want %q (B's runtime must be preserved)", got, "B_RUNTIME")
	}
	if got := conf["ssh/host"]; got != "" {
		t.Errorf("ssh/host = %q, want empty (CWD's .madock must NOT be merged for a foreign project)", got)
	}
	if got := conf["path"]; got == tmpRun {
		t.Errorf("path = %q, must not be silently set to the caller's CWD", got)
	}
}

// TestGetProjectConfigOnly_CurrentProjectUsesCWD verifies backward compatibility:
// for the *current* project (no `path` in runtime), the CWD fallback still applies
// and the release .madock/config.xml is merged.
func TestGetProjectConfigOnly_CurrentProjectUsesCWD(t *testing.T) {
	tmpExec := t.TempDir()
	tmpRun := t.TempDir()
	t.Setenv("MADOCK_EXEC_DIR", tmpExec)
	t.Setenv("MADOCK_RUN_DIR", tmpRun)

	if err := os.MkdirAll(filepath.Join(tmpRun, ".madock"), 0o755); err != nil {
		t.Fatal(err)
	}
	SaveInFile(filepath.Join(tmpRun, ".madock", "config.xml"), map[string]string{"ssh/host": "FROM_CWD"}, "default")

	if err := os.MkdirAll(filepath.Join(tmpExec, "projects", "A"), 0o755); err != nil {
		t.Fatal(err)
	}
	SaveInFile(filepath.Join(tmpExec, "projects", "A", "config.xml"), map[string]string{"db/database": "A_RUNTIME"}, "default")

	CleanCache()
	SetProjectNameResolver(func() string { return "A" })
	t.Cleanup(func() { SetProjectNameResolver(nil); CleanCache() })

	conf := GetProjectConfigOnly("A")

	if got := conf["db/database"]; got != "A_RUNTIME" {
		t.Errorf("db/database = %q, want %q", got, "A_RUNTIME")
	}
	if got := conf["ssh/host"]; got != "FROM_CWD" {
		t.Errorf("ssh/host = %q, want %q (current project's release config must merge)", got, "FROM_CWD")
	}
	if got := conf["path"]; got != tmpRun {
		t.Errorf("path = %q, want %q (current project keeps CWD fallback)", got, tmpRun)
	}
}
