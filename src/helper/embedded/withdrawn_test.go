package embedded

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Extraction only ever added and overwrote, so a template a release withdrew
// stayed on the machine for good — and the resolver goes on applying it. The
// result then depends on the history of the installation rather than on the
// version it reports.
func TestRemoveWithdrawn(t *testing.T) {
	dir := t.TempDir()

	write(t, filepath.Join(dir, manifestFile), "docker/a.yml\ndocker/gone.yml\n")
	write(t, filepath.Join(dir, "docker", "a.yml"), "kept")
	write(t, filepath.Join(dir, "docker", "gone.yml"), "withdrawn")

	removeWithdrawn(dir, map[string]bool{"docker/a.yml": true})

	if !exists(filepath.Join(dir, "docker", "a.yml")) {
		t.Error("a file this extraction wrote was removed")
	}
	if exists(filepath.Join(dir, "docker", "gone.yml")) {
		t.Error("a withdrawn file survived")
	}
}

// The rule that makes this safe: only files this mechanism itself wrote are
// removed, and it knows which those are from its own manifest. madock-pro
// extracts its own platform templates into the same tree, so a sweep of
// everything-not-in-the-embed would delete them.
func TestRemoveWithdrawn_LeavesForeignFilesAlone(t *testing.T) {
	dir := t.TempDir()

	write(t, filepath.Join(dir, manifestFile), "docker/a.yml\n")
	write(t, filepath.Join(dir, "docker", "a.yml"), "ours")
	write(t, filepath.Join(dir, "docker", "packeton", "docker-compose.yml"), "pro's")
	write(t, filepath.Join(dir, "docker", "notes.txt"), "somebody's")

	removeWithdrawn(dir, map[string]bool{})

	for _, keep := range []string{
		filepath.Join(dir, "docker", "packeton", "docker-compose.yml"),
		filepath.Join(dir, "docker", "notes.txt"),
	} {
		if !exists(keep) {
			t.Errorf("a file this mechanism never wrote was removed: %s", keep)
		}
	}
	if exists(filepath.Join(dir, "docker", "a.yml")) {
		t.Error("a file in the manifest and not in this extraction survived")
	}
}

// An installation that has never written a manifest — every installation before
// this version — has nothing removed. Orphans predating it need one clean by
// hand, which is stated rather than guessed at.
func TestRemoveWithdrawn_NoManifestRemovesNothing(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "docker", "orphan.yml"), "from before")

	removeWithdrawn(dir, map[string]bool{})

	if !exists(filepath.Join(dir, "docker", "orphan.yml")) {
		t.Error("a file was removed on an installation with no manifest")
	}
}

// The manifest is a file on disk and this deletes what it names, so a path that
// climbs out of the tree is refused.
func TestRemoveWithdrawn_RefusesToClimbOut(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(dir, "outside.txt")
	write(t, outside, "not ours")

	inner := filepath.Join(dir, "install")
	write(t, filepath.Join(inner, manifestFile), "../outside.txt\n/etc/passwd\n")

	removeWithdrawn(inner, map[string]bool{})

	if !exists(outside) {
		t.Error("a relative path climbed out of the installation and deleted a file")
	}
}

// And the manifest round-trips, or the next run removes everything it wrote.
func TestWriteManifest(t *testing.T) {
	dir := t.TempDir()
	writeManifest(dir, map[string]bool{"docker/b.yml": true, "docker/a.yml": true})

	body, err := os.ReadFile(filepath.Join(dir, manifestFile))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "docker/a.yml\ndocker/b.yml\n" {
		t.Errorf("manifest is %q", body)
	}
}
