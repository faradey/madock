package configs

import (
	"os"
	"path/filepath"
	"strings"
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

// registerProject writes a registry entry recording a source path.
func registerProject(t *testing.T, execDir, name, path string) string {
	t.Helper()

	runtime := filepath.Join(execDir, "projects", name, "config.xml")
	if err := os.MkdirAll(filepath.Dir(runtime), 0o755); err != nil {
		t.Fatal(err)
	}
	SaveInFile(runtime, map[string]string{"path": path}, "default")

	return runtime
}

// The ghost, at the moment it would be created: madock run inside a Deployer
// release of an application that is registered already. Nothing about the
// directory is wrong — it exists, it has a .madock/config.xml, and the name it
// would take is the name of the release symlink.
func TestRegistrationRefusal_InsideARegisteredProject(t *testing.T) {
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	execDir := filepath.Join(base, "exec")
	app := filepath.Join(base, "ops-console")
	release := filepath.Join(app, "releases", "847")
	if err := os.MkdirAll(release, 0o755); err != nil {
		t.Fatal(err)
	}
	current := filepath.Join(app, "current")
	if err := os.Symlink(release, current); err != nil {
		t.Fatal(err)
	}

	t.Setenv("MADOCK_EXEC_DIR", execDir)
	CleanCache()
	t.Cleanup(CleanCache)

	registerProject(t, execDir, "ops-console", app)

	refusal := RegistrationRefusal(current)
	if len(refusal) == 0 {
		t.Fatal("a release directory of a registered application must not become a project of its own")
	}
	if !strings.Contains(strings.Join(refusal, "\n"), "ops-console") {
		t.Errorf("the refusal has to name the project it is standing in, got:\n%s", strings.Join(refusal, "\n"))
	}
	if !strings.Contains(strings.Join(refusal, "\n"), release) {
		t.Errorf("the refusal has to say where the path resolves, got:\n%s", strings.Join(refusal, "\n"))
	}
}

// The other half, and it bites where nothing else is registered: a symlink
// records a path that the next deploy repoints, so the entry comes to describe a
// directory nobody chose.
func TestRegistrationRefusal_SymlinkedDirectory(t *testing.T) {
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	execDir := filepath.Join(base, "exec")
	release := filepath.Join(base, "releases", "12")
	if err := os.MkdirAll(release, 0o755); err != nil {
		t.Fatal(err)
	}
	current := filepath.Join(base, "current")
	if err := os.Symlink(release, current); err != nil {
		t.Fatal(err)
	}

	t.Setenv("MADOCK_EXEC_DIR", execDir)
	CleanCache()
	t.Cleanup(CleanCache)

	if refusal := RegistrationRefusal(current); len(refusal) == 0 {
		t.Fatal("registering through a symlink records a path that moves; it has to be refused")
	}
}

// And an ordinary directory registers, which is the case that must not be made
// harder: a guard that also refuses the normal path is worse than no guard.
func TestRegistrationRefusal_PlainDirectoryIsFine(t *testing.T) {
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	execDir := filepath.Join(base, "exec")
	project := filepath.Join(base, "shop")
	sibling := filepath.Join(base, "shop2")
	for _, dir := range []string{project, sibling} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	t.Setenv("MADOCK_EXEC_DIR", execDir)
	CleanCache()
	t.Cleanup(CleanCache)

	registerProject(t, execDir, "shop", project)

	// A sibling sharing a prefix is not inside anything.
	if refusal := RegistrationRefusal(sibling); len(refusal) > 0 {
		t.Errorf("a sibling directory was refused:\n%s", strings.Join(refusal, "\n"))
	}
	// And the project itself is not inside itself.
	if refusal := RegistrationRefusal(project); len(refusal) > 0 {
		t.Errorf("the project's own directory was refused:\n%s", strings.Join(refusal, "\n"))
	}
}

// The guard fires only where a new entry would be created. An installation that
// already carries one of these entries keeps working — this stops the registry
// growing, it does not break the machines that already have the problem.
func TestIsHasConfig_AlreadyRegisteredNestedProjectStillWorks(t *testing.T) {
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	execDir := filepath.Join(base, "exec")
	app := filepath.Join(base, "app")
	release := filepath.Join(app, "releases", "5")
	if err := os.MkdirAll(filepath.Join(release, ".madock"), 0o755); err != nil {
		t.Fatal(err)
	}
	SaveInFile(filepath.Join(release, ".madock", "config.xml"), map[string]string{"platform": "magento2"}, "default")

	t.Setenv("MADOCK_EXEC_DIR", execDir)
	t.Setenv("MADOCK_RUN_DIR", release)
	CleanCache()
	t.Cleanup(CleanCache)

	registerProject(t, execDir, "app", app)
	runtime := registerProject(t, execDir, "ghost", "")

	// Would exit(1) if the guard fired on an entry that exists already.
	if !IsHasConfig("ghost") {
		t.Fatal("an existing entry should still be read")
	}
	if got := persistedPath(t, runtime); got != release {
		t.Errorf("path after heal = %q, want %q", got, release)
	}
}
