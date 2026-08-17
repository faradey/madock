package embedded

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"testing/fstest"
)

// The defect this exists for: the installation and the working copy are the
// same directory in a source install, so extracting a build-time snapshot over
// it reverts whatever was edited since the binary was built. Measured — three
// edited templates disappeared mid-session, and a test then passed against the
// reverted files.
func TestExtractIfNeeded_LeavesASourceCheckoutAlone(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MADOCK_EXEC_DIR", dir)

	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// The signal is the embed declaration in the template tree, not go.mod —
	// see isSourceCheckout for why the module file was the wrong question.
	if err := os.MkdirAll(filepath.Join(dir, "docker"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(sourceSentinel)), []byte("package dockerassets\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	edited := filepath.Join(dir, "docker", "snippets", "probe.yml")
	if err := os.MkdirAll(filepath.Dir(edited), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(edited, []byte("edited by hand\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	DockerFS = fstest.MapFS{
		"snippets/probe.yml": {Data: []byte("from the binary\n")},
	}
	t.Cleanup(func() { DockerFS = nil })

	ExtractIfNeeded("9.9.9")

	body, err := os.ReadFile(edited)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "edited by hand\n" {
		t.Fatalf("the working copy was overwritten by the embedded snapshot: %q", string(body))
	}

	if _, err := os.Stat(filepath.Join(dir, ".embedded_version")); err == nil {
		t.Error("a source checkout should not be stamped with an embedded version either")
	}
}

// The other half of the same question, and the one go.mod got wrong: an
// installation can be a Go module and still have no other way to get its
// templates. madock-pro is a clone with go.mod at its root, but docker/ is in
// its .gitignore on purpose — the assets belong to the imported module and are
// extracted at runtime. Testing go.mod therefore skipped extraction in the one
// installation where nothing else delivers the tree, and it froze: measured
// 2026-08-17, .embedded_version said 3.6.7 while the module was 3.9.3, and 47
// templates were still in the pre-3.9.1 syntax. A customer install, being a
// bare binary, was fine — so the breakage was confined to the installation pro
// is developed and tested on.
func TestExtractIfNeeded_FillsAModuleThatDoesNotShipItsTemplates(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MADOCK_EXEC_DIR", dir)

	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// What an earlier binary left behind, with the stamp to match.
	stale := filepath.Join(dir, "docker", "snippets", "probe.yml")
	if err := os.MkdirAll(filepath.Dir(stale), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stale, []byte("from an old binary\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".embedded_version"), []byte("3.6.7"), 0o644); err != nil {
		t.Fatal(err)
	}

	DockerFS = fstest.MapFS{
		"snippets/probe.yml": {Data: []byte("from the binary\n")},
	}
	t.Cleanup(func() { DockerFS = nil })

	ExtractIfNeeded("9.9.9")

	body, err := os.ReadFile(stale)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "from the binary\n" {
		t.Fatalf("the stale tree was left in place: %q", string(body))
	}

	stamp, err := os.ReadFile(filepath.Join(dir, ".embedded_version"))
	if err != nil || string(stamp) != "9.9.9" {
		t.Fatalf("the version stamp was not updated: %q, %v", string(stamp), err)
	}
}

// The sentinel is a path, so a rename would turn the guard off silently and the
// reverted-edits defect would come back with nothing failing. This is the only
// test that knows where the repository root is, and that is the point.
func TestSourceSentinelIsWhereTheGuardExpectsIt(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Skip("no caller information")
	}
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(sourceSentinel))); err != nil {
		t.Fatalf("%s is gone from the repository, so isSourceCheckout now answers false "+
			"for madock's own tree and a development binary will revert edited templates again: %v",
			sourceSentinel, err)
	}
}

// A binary installation has nothing but the binary, so the templates have to
// come out of it — that path must keep working.
func TestExtractIfNeeded_StillFillsABinaryInstallation(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MADOCK_EXEC_DIR", dir)

	DockerFS = fstest.MapFS{
		"snippets/probe.yml": {Data: []byte("from the binary\n")},
	}
	t.Cleanup(func() { DockerFS = nil })

	ExtractIfNeeded("9.9.9")

	body, err := os.ReadFile(filepath.Join(dir, "docker", "snippets", "probe.yml"))
	if err != nil {
		t.Fatalf("the templates were not extracted: %v", err)
	}
	if string(body) != "from the binary\n" {
		t.Fatalf("unexpected content: %q", string(body))
	}

	stamp, err := os.ReadFile(filepath.Join(dir, ".embedded_version"))
	if err != nil || string(stamp) != "9.9.9" {
		t.Fatalf("the version stamp was not written: %q, %v", string(stamp), err)
	}
}
