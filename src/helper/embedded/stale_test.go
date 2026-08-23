package embedded

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

// sourceTree makes a temp directory look like a clone: the embed declaration is
// what says the templates here are git's to deliver, so it is what makes drift
// possible at all.
func sourceTree(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	t.Setenv("MADOCK_EXEC_DIR", dir)

	if err := os.MkdirAll(filepath.Join(dir, "docker"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(sourceSentinel)), []byte("package dockerassets\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	return dir
}

func writeTemplate(t *testing.T, dir, rel, body string) {
	t.Helper()

	path := filepath.Join(dir, "docker", filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// The incident this exists for, in miniature: the binary is older than the tree,
// the php snippets have been reorganised under it, and the only way anyone found
// out was a rebuild that had already destroyed the containers.
func TestTemplateDrift_SeesABinaryOlderThanTheTree(t *testing.T) {
	dir := sourceTree(t)

	writeTemplate(t, dir, "snippets/dockerfile/php/header", "the new layout\n")
	writeTemplate(t, dir, "general/docker-compose.yml", "edited\n")

	DockerFS = fstest.MapFS{
		// Where the snippet used to live, and where this binary still looks.
		"snippets/dockerfile/php/nodejs": {Data: []byte("the old layout\n")},
		"general/docker-compose.yml":     {Data: []byte("as built\n")},
		"magento2/docker-compose.yml":    {Data: []byte("unchanged\n")},
	}
	t.Cleanup(func() { DockerFS = nil })

	writeTemplate(t, dir, "magento2/docker-compose.yml", "unchanged\n")

	drift := TemplateDrift()
	if drift == nil {
		t.Fatal("no drift reported against a tree the binary was not built from")
	}

	if len(drift.Missing) != 1 || drift.Missing[0] != "snippets/dockerfile/php/nodejs" {
		t.Errorf("the withdrawn template was not reported as missing: %v", drift.Missing)
	}
	if len(drift.Extra) != 1 || drift.Extra[0] != "snippets/dockerfile/php/header" {
		t.Errorf("the template the binary has never heard of was not reported: %v", drift.Extra)
	}
	if len(drift.Changed) != 1 || drift.Changed[0] != "general/docker-compose.yml" {
		t.Errorf("the edited template was not reported as changed: %v", drift.Changed)
	}
}

func TestTemplateDrift_SilentWhenTheyAgree(t *testing.T) {
	dir := sourceTree(t)

	writeTemplate(t, dir, "general/docker-compose.yml", "as built\n")

	DockerFS = fstest.MapFS{
		"general/docker-compose.yml": {Data: []byte("as built\n")},
	}
	t.Cleanup(func() { DockerFS = nil })

	if drift := TemplateDrift(); drift != nil {
		t.Fatalf("a binary built from these very templates was called stale: %+v", drift)
	}
}

// Where extraction owns the tree the two cannot disagree, and a warning would be
// noise on every customer installation and on madock-pro's — neither of which
// has a compiler to act on it.
func TestTemplateDrift_SaysNothingWhereExtractionOwnsTheTree(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MADOCK_EXEC_DIR", dir)

	// No docker/embed.go: this is an extracted tree, not a clone.
	if err := os.MkdirAll(filepath.Join(dir, "docker", "general"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "docker", "general", "docker-compose.yml"), []byte("from an old binary\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	DockerFS = fstest.MapFS{
		"general/docker-compose.yml": {Data: []byte("from the binary\n")},
	}
	t.Cleanup(func() { DockerFS = nil })

	if drift := TemplateDrift(); drift != nil {
		t.Fatalf("drift reported where extraction is responsible for the tree: %+v", drift)
	}
}

// Finder writes these into any directory somebody has looked at, so counting
// them would make the check warn on every Mac it runs on — including this one,
// where docker/ already carries them.
func TestTemplateDrift_IgnoresDSStore(t *testing.T) {
	dir := sourceTree(t)

	writeTemplate(t, dir, "general/docker-compose.yml", "as built\n")
	writeTemplate(t, dir, "general/.DS_Store", "\x00\x00")

	DockerFS = fstest.MapFS{
		"general/docker-compose.yml": {Data: []byte("as built\n")},
	}
	t.Cleanup(func() { DockerFS = nil })

	if drift := TemplateDrift(); drift != nil {
		t.Fatalf("a Finder leftover was reported as drift: %+v", drift)
	}
}

// The tests beside the templates are source, not assets — they are in no
// //go:embed pattern and never will be, so reporting them would make the check
// permanently wrong about madock's own tree.
func TestTemplateDrift_IgnoresSourceBesideTheTemplates(t *testing.T) {
	dir := sourceTree(t)

	writeTemplate(t, dir, "general/docker-compose.yml", "as built\n")
	writeTemplate(t, dir, "embed_test.go", "package dockerassets\n")

	DockerFS = fstest.MapFS{
		"general/docker-compose.yml": {Data: []byte("as built\n")},
	}
	t.Cleanup(func() { DockerFS = nil })

	if drift := TemplateDrift(); drift != nil {
		t.Fatalf("source at the root of docker/ was reported as drift: %+v", drift)
	}
}
