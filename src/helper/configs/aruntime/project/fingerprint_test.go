package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/faradey/madock/v3/src/helper/paths"
)

// stack writes a fake generated runtime dir for a project and returns its path.
func stack(t *testing.T, projectName string, files map[string]string) string {
	t.Helper()
	pp := paths.NewProjectPaths(projectName)
	root := pp.RuntimeDir()
	for name, content := range files {
		full := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestFingerprintChangesWithContent(t *testing.T) {
	t.Setenv("MADOCK_EXEC_DIR", t.TempDir())

	stack(t, "demo", map[string]string{
		"docker-compose.yml": "services: {}",
		"ctx/php.Dockerfile": "FROM php:8.3",
	})
	before := Fingerprint("demo")
	if before == "" {
		t.Fatal("Fingerprint returned nothing for a generated stack")
	}

	stack(t, "demo", map[string]string{"ctx/php.Dockerfile": "FROM php:8.4"})
	after := Fingerprint("demo")

	if before == after {
		t.Error("Fingerprint did not change after the Dockerfile did")
	}
}

// Regenerating from an unchanged config rewrites the same bytes. If that moved
// the fingerprint, every single start would recreate the containers.
func TestFingerprintStableAcrossIdenticalRenders(t *testing.T) {
	t.Setenv("MADOCK_EXEC_DIR", t.TempDir())

	stack(t, "demo", map[string]string{
		"docker-compose.yml": "services: {}",
		"ctx/php.Dockerfile": "FROM php:8.3",
	})
	first := Fingerprint("demo")

	// Same content written again, as MakeConf does on every up.
	stack(t, "demo", map[string]string{
		"docker-compose.yml": "services: {}",
		"ctx/php.Dockerfile": "FROM php:8.3",
	})

	if second := Fingerprint("demo"); first != second {
		t.Error("Fingerprint moved although the generated files are identical")
	}
}

// The runtime dir holds symlinks to the project source, ~/.composer and ~/.ssh.
// Following them would hash the whole working tree, so every source edit would
// look like a stack change and force a rebuild on the next start.
func TestFingerprintIgnoresSymlinks(t *testing.T) {
	t.Setenv("MADOCK_EXEC_DIR", t.TempDir())

	root := stack(t, "demo", map[string]string{"docker-compose.yml": "services: {}"})
	before := Fingerprint("demo")

	target := filepath.Join(t.TempDir(), "src")
	if err := os.MkdirAll(target, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "index.php"), []byte("<?php"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(root, "src")); err != nil {
		t.Fatal(err)
	}

	if after := Fingerprint("demo"); before != after {
		t.Error("Fingerprint followed a symlink out of the runtime dir")
	}
}

func TestNeedsRecreateAfterAChange(t *testing.T) {
	t.Setenv("MADOCK_EXEC_DIR", t.TempDir())

	stack(t, "demo", map[string]string{"ctx/php.Dockerfile": "FROM php:8.3"})
	RecordApplied("demo")

	if NeedsRecreate("demo") {
		t.Error("NeedsRecreate is true right after recording the same stack")
	}

	stack(t, "demo", map[string]string{"ctx/php.Dockerfile": "FROM php:8.4"})

	if !NeedsRecreate("demo") {
		t.Error("NeedsRecreate missed a changed Dockerfile")
	}
}

// A project that predates the fingerprint has no record. Treating that as a
// change would rebuild every project on the first start after an upgrade.
func TestNeedsRecreateAdoptsAnUnknownStack(t *testing.T) {
	t.Setenv("MADOCK_EXEC_DIR", t.TempDir())

	stack(t, "demo", map[string]string{"ctx/php.Dockerfile": "FROM php:8.3"})

	if NeedsRecreate("demo") {
		t.Error("NeedsRecreate forced a rebuild for a stack it had never seen")
	}
	// ...and it adopted it, so the next call is quiet too.
	if NeedsRecreate("demo") {
		t.Error("NeedsRecreate did not record the stack it adopted")
	}
}
