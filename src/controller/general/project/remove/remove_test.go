package remove

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/faradey/madock/v4/src/helper/configs"
)

// registryWith builds an installation holding one project entry, with the
// recorded source path pointing where the caller says.
func registryWith(t *testing.T, name, sourcePath string) string {
	t.Helper()

	execDir := t.TempDir()
	dir := filepath.Join(execDir, "projects", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	config := `<?xml version="1.0" encoding="UTF-8"?>
<config>
    <scopes>
        <default>
            <path>` + sourcePath + `</path>
        </default>
    </scopes>
</config>
`
	if err := os.WriteFile(filepath.Join(dir, "config.xml"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}

	return execDir
}

// The orphan is the case the whole thing exists for: an entry whose source
// directory is gone. project:list has been able to name those since 3.8.50, and
// nothing could remove them — the command asked the working directory who the
// project was, and for an orphan there is no directory to stand in.
func TestListProjects_NamesTheOrphan(t *testing.T) {
	execDir := registryWith(t, "ghost", filepath.Join(t.TempDir(), "gone"))

	entries := configs.ListProjectsIn(execDir)
	if len(entries) != 1 {
		t.Fatalf("expected one entry, got %d", len(entries))
	}
	if entries[0].State != configs.ProjectMissingSource {
		t.Fatalf("expected the entry to be an orphan, got state %q", entries[0].State)
	}
}

// A project whose source is still there must not be removable from somewhere
// else: removeProject ends with RemoveAll on the working directory, so that
// would take the caller's directory with it.
func TestListProjects_LiveProjectIsNotAnOrphan(t *testing.T) {
	source := t.TempDir()
	execDir := registryWith(t, "alive", source)

	entries := configs.ListProjectsIn(execDir)
	if entries[0].State != configs.ProjectOk {
		t.Fatalf("a project with its source present should be ok, got %q", entries[0].State)
	}
}

// The command has to be reachable outside a project, or --name can never run:
// the scope check refuses project commands elsewhere, and it runs before the
// handler that would decide.
func TestCommandIsReachableOutsideAProject(t *testing.T) {
	if !definition().Global {
		t.Fatal("project:remove is refused outside a project, so an orphan cannot be removed at all")
	}
}
