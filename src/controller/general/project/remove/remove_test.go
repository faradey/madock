package remove

import (
	"os"
	"path/filepath"
	"strings"
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

// deployerLayout builds what a Deployer installation looks like: numbered
// releases and a `current` symlink pointing at one of them. The base is resolved
// first — on macOS t.TempDir() sits under /var, which is itself a symlink, and a
// test that ignored that would be asserting about the machine rather than about
// the code.
func deployerLayout(t *testing.T) (base, release, current string) {
	t.Helper()

	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	release = filepath.Join(base, "releases", "847")
	if err := os.MkdirAll(release, 0o755); err != nil {
		t.Fatal(err)
	}

	current = filepath.Join(base, "current")
	if err := os.Symlink(release, current); err != nil {
		t.Fatal(err)
	}

	return base, release, current
}

// The warning has to name the release directory, because that is what RemoveAll
// takes when the shell handed over a physical path — `cd -P`, or any shell that
// resolves. Measured: the same command with the link kept in the path removes
// the link alone.
func TestRemovalTargetLines_NamesTheReleaseBehindCurrent(t *testing.T) {
	_, release, current := deployerLayout(t)

	// As `cd -P` leaves it: the physical path, no link in it.
	lines := removalTargetLines(release)
	if len(lines) != 1 || lines[0] != release {
		t.Fatalf("a physical path should be printed as itself, got %#v", lines)
	}

	// As a plain `cd` leaves it: the link itself.
	lines = removalTargetLines(current)
	if len(lines) != 2 {
		t.Fatalf("a symlinked working directory should print what it resolves to, got %#v", lines)
	}
	if lines[0] != current {
		t.Fatalf("the first line should be the path as typed, got %q", lines[0])
	}
	if !strings.Contains(lines[1], release) {
		t.Fatalf("the second line should name the release directory, got %q", lines[1])
	}
	if !strings.Contains(lines[1], "stays") {
		t.Fatalf("a symlinked directory loses the link only; the line should say so, got %q", lines[1])
	}
}

// A directory reached through a symlinked parent is the dangerous half: the path
// is not a link, so RemoveAll goes straight through into the real directory.
func TestRemovalTargetLines_ResolvesASymlinkedParent(t *testing.T) {
	_, release, current := deployerLayout(t)

	inner := filepath.Join(release, "pub")
	if err := os.MkdirAll(inner, 0o755); err != nil {
		t.Fatal(err)
	}

	lines := removalTargetLines(filepath.Join(current, "pub"))
	if len(lines) != 2 {
		t.Fatalf("expected the resolved path to be named, got %#v", lines)
	}
	if !strings.Contains(lines[1], inner) {
		t.Fatalf("the resolved path should be the real directory %q, got %q", inner, lines[1])
	}
	if !strings.Contains(lines[1], "deleted") {
		t.Fatalf("the line should say this is the directory that gets deleted, got %q", lines[1])
	}
}

// An ordinary project directory prints one line and no explanation: the extra
// line exists to be noticed, and printing it always is how it stops being read.
func TestRemovalTargetLines_PlainDirectoryStaysOneLine(t *testing.T) {
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	lines := removalTargetLines(dir)
	if len(lines) != 1 || lines[0] != dir {
		t.Fatalf("expected a single line naming the directory, got %#v", lines)
	}
}

// The ban on destructive commands is what madock-pro ships on servers, and it
// refused the one removal that cannot destroy anything — which is how three
// registry entries pointing into live releases stayed on a production host with
// no way to remove them.
func TestRegistryOnlyAllowed_UnderTheBan(t *testing.T) {
	cases := []struct {
		state string
		want  bool
		why   string
	}{
		{configs.ProjectNestedPath, true, "the path is inside another project, so the entry owns nothing"},
		{configs.ProjectMissingSource, true, "the source directory is gone"},
		{configs.ProjectBrokenLink, true, "the entry resolves to nothing at all"},
		{configs.ProjectOk, false, "a healthy project: its record is its madock configuration"},
		{configs.ProjectNoPath, false, "legacy entry of a project that may still exist — not a thing to guess about"},
	}

	for _, c := range cases {
		if got := registryOnlyAllowed(c.state, false); got != c.want {
			t.Errorf("state %q under the ban = %v, want %v — %s", c.state, got, c.want, c.why)
		}
		// With the ban lifted every state is the caller's business.
		if !registryOnlyAllowed(c.state, true) {
			t.Errorf("state %q was refused on an installation that allows destructive commands", c.state)
		}
	}
}
