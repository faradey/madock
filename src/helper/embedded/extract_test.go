package embedded

import (
	"os"
	"path/filepath"
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
