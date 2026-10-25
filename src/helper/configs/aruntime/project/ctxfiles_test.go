package project

import (
	"os"
	"path/filepath"
	"testing"
)

// isolateCtxFiles empties the registry for one test and puts back what was
// registered — madock's own madockCA.crt among it — when the test ends.
func isolateCtxFiles(t *testing.T) {
	t.Helper()
	prev := ctxFiles
	ctxFiles = nil
	t.Cleanup(func() { ctxFiles = prev })
}

// With nothing registered the seam writes nothing.
func TestWithNoProviderTheCtxDirectoryIsUnchanged(t *testing.T) {
	isolateCtxFiles(t)

	dir := t.TempDir()
	writeCtxFiles("someproject", dir)

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading %s: %v", dir, err)
	}
	if len(entries) != 0 {
		t.Errorf("the seam created files with nothing registered: %v", entries)
	}
}

func TestAProviderWritesItsFile(t *testing.T) {
	isolateCtxFiles(t)

	RegisterCtxFile(CtxFile{
		Name:   "madock-logs.conf",
		Render: func(string) (string, bool) { return "access_log /var/log/madock/x.log;\n", true },
	})

	dir := t.TempDir()
	writeCtxFiles("someproject", dir)

	body, err := os.ReadFile(filepath.Join(dir, "madock-logs.conf"))
	if err != nil {
		t.Fatalf("the provider's file was not written: %v", err)
	}
	if string(body) != "access_log /var/log/madock/x.log;\n" {
		t.Errorf("content is %q", body)
	}
}

// "This file should not exist" has to remove one that does, and that is not
// tidiness: the mount that uses it is conditional on the same setting, so a
// leftover points a container at something that is no longer there — and nginx
// refuses to start over a path it cannot open.
func TestAProviderCanTakeItsFileAway(t *testing.T) {
	isolateCtxFiles(t)

	dir := t.TempDir()
	stale := filepath.Join(dir, "madock-logs.conf")
	if err := os.WriteFile(stale, []byte("from an earlier configuration\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	RegisterCtxFile(CtxFile{
		Name:   "madock-logs.conf",
		Render: func(string) (string, bool) { return "", false },
	})
	writeCtxFiles("someproject", dir)

	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Errorf("the stale file survived: %v", err)
	}
}
