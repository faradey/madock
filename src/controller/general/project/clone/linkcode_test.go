package clone

import (
	"os"
	"path/filepath"
	"testing"
)

// The clone's directory is made empty before the engine clones the storage;
// with the code in that storage it becomes a link to the clone's copy.
func TestAnEmptyCloneDirectoryBecomesALinkToTheCode(t *testing.T) {
	root := t.TempDir()
	clonePath, code := filepath.Join(root, "shop-copy"), filepath.Join(root, "storage", "code")
	if err := os.MkdirAll(clonePath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(code, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := linkCode(clonePath, code); err != nil {
		t.Fatal(err)
	}
	if target, err := os.Readlink(clonePath); err != nil || target != code {
		t.Fatalf("clone directory is not a link to the code: %q %v", target, err)
	}
}

// Something already in the directory would be hidden behind the link.
func TestACloneDirectoryWithFilesIsNotReplaced(t *testing.T) {
	root := t.TempDir()
	clonePath := filepath.Join(root, "shop-copy")
	if err := os.MkdirAll(clonePath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(clonePath, "index.php"), []byte("<?php"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := linkCode(clonePath, filepath.Join(root, "code")); err == nil {
		t.Fatal("a directory holding a file was replaced by a link")
	}
	if _, err := os.Stat(filepath.Join(clonePath, "index.php")); err != nil {
		t.Fatalf("the file is gone: %v", err)
	}
}
