package configs

import (
	"os"
	"path/filepath"
	"testing"
)

func persistedPath(t *testing.T, runtimeConfigPath string) string {
	t.Helper()
	raw := ParseXmlFile(runtimeConfigPath)
	return getConfigByScope(raw, "default")["path"]
}

// TestIsHasConfig_SelfHealsMissingPath: a legacy project whose runtime config has
// no `path` key gets it backfilled from the current source dir on the next call.
func TestIsHasConfig_SelfHealsMissingPath(t *testing.T) {
	tmpExec := t.TempDir()
	tmpRun := t.TempDir()
	t.Setenv("MADOCK_EXEC_DIR", tmpExec)
	t.Setenv("MADOCK_RUN_DIR", tmpRun)

	if err := os.MkdirAll(filepath.Join(tmpRun, ".madock"), 0o755); err != nil {
		t.Fatal(err)
	}
	SaveInFile(filepath.Join(tmpRun, ".madock", "config.xml"), map[string]string{"platform": "magento2"}, "default")

	runtime := filepath.Join(tmpExec, "projects", "LEGACY", "config.xml")
	if err := os.MkdirAll(filepath.Dir(runtime), 0o755); err != nil {
		t.Fatal(err)
	}
	SaveInFile(runtime, map[string]string{"db/database": "legacy_db"}, "default")

	CleanCache()
	t.Cleanup(CleanCache)

	if got := persistedPath(t, runtime); got != "" {
		t.Fatalf("precondition: path already set = %q", got)
	}

	IsHasConfig("LEGACY")

	if got := persistedPath(t, runtime); got != tmpRun {
		t.Errorf("path after heal = %q, want %q", got, tmpRun)
	}
}

// TestIsHasConfig_NoHealWithoutInProjectConfig: without a .madock/config.xml in
// CWD we cannot trust GetRunDirPath as this project's source, so do not write path.
func TestIsHasConfig_NoHealWithoutInProjectConfig(t *testing.T) {
	tmpExec := t.TempDir()
	tmpRun := t.TempDir()
	t.Setenv("MADOCK_EXEC_DIR", tmpExec)
	t.Setenv("MADOCK_RUN_DIR", tmpRun)

	runtime := filepath.Join(tmpExec, "projects", "NOMADOCK", "config.xml")
	if err := os.MkdirAll(filepath.Dir(runtime), 0o755); err != nil {
		t.Fatal(err)
	}
	SaveInFile(runtime, map[string]string{"db/database": "x"}, "default")

	CleanCache()
	t.Cleanup(CleanCache)

	IsHasConfig("NOMADOCK")

	if got := persistedPath(t, runtime); got != "" {
		t.Errorf("path = %q, want empty (no .madock in CWD must not heal)", got)
	}
}

// TestIsHasConfig_KeepsExistingPath: an already-recorded path is left untouched.
func TestIsHasConfig_KeepsExistingPath(t *testing.T) {
	tmpExec := t.TempDir()
	tmpRun := t.TempDir()
	t.Setenv("MADOCK_EXEC_DIR", tmpExec)
	t.Setenv("MADOCK_RUN_DIR", tmpRun)

	if err := os.MkdirAll(filepath.Join(tmpRun, ".madock"), 0o755); err != nil {
		t.Fatal(err)
	}
	SaveInFile(filepath.Join(tmpRun, ".madock", "config.xml"), map[string]string{"platform": "magento2"}, "default")

	runtime := filepath.Join(tmpExec, "projects", "HASPATH", "config.xml")
	if err := os.MkdirAll(filepath.Dir(runtime), 0o755); err != nil {
		t.Fatal(err)
	}
	SaveInFile(runtime, map[string]string{"path": "/original/path", "db/database": "x"}, "default")

	CleanCache()
	t.Cleanup(CleanCache)

	IsHasConfig("HASPATH")

	if got := persistedPath(t, runtime); got != "/original/path" {
		t.Errorf("path = %q, want %q (must not overwrite existing)", got, "/original/path")
	}
}
