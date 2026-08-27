package cron

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// On a plain checkout the path madock reads is the file a person edits, so the
// message is short and names it once.
func TestOverrideMessageNamesTheProjectFile(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".madock")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("creating %s: %v", dir, err)
	}
	own := filepath.Join(dir, "config.xml")

	lines := whereToEditLines(dir, own)
	joined := strings.Join(lines, "\n")

	if !strings.Contains(joined, own) {
		t.Errorf("the message does not name the file that overrides the setting:\n%s", joined)
	}
	if strings.Contains(joined, "release") {
		t.Errorf("a checkout was described as a release:\n%s", joined)
	}
	if strings.Count(joined, own) != 1 {
		t.Errorf("the path is repeated; say it once:\n%s", joined)
	}
}

// Where deployer manages the project, `.madock` is a symlink into the current
// release. The path is still the one madock reads, and it is emphatically not
// the one to edit: the next deploy restores the file from the repository, so an
// edit there is right until a release undoes it with nothing said.
func TestOverrideMessageSendsADeployedProjectToItsRepository(t *testing.T) {
	root := t.TempDir()
	release := filepath.Join(root, "releases", "46", ".madock")
	if err := os.MkdirAll(release, 0o755); err != nil {
		t.Fatalf("creating %s: %v", release, err)
	}
	if err := os.WriteFile(filepath.Join(release, "config.xml"), []byte("<config/>"), 0o644); err != nil {
		t.Fatalf("writing the release config: %v", err)
	}
	if err := os.Symlink(filepath.Join("releases", "46", ".madock"), filepath.Join(root, ".madock")); err != nil {
		t.Fatalf("linking .madock: %v", err)
	}

	dir := filepath.Join(root, ".madock")
	lines := whereToEditLines(dir, filepath.Join(dir, "config.xml"))
	joined := strings.Join(lines, "\n")

	if !strings.Contains(joined, "repository") {
		t.Errorf("a deployed project was not sent to its repository:\n%s", joined)
	}
	if !strings.Contains(joined, "releases/46") {
		t.Errorf("the message does not say where the path actually leads:\n%s", joined)
	}
	if !strings.Contains(joined, "undone by the next release") {
		t.Errorf("the message does not say what happens to an edit made here:\n%s", joined)
	}
}

// A broken link still answers the half that matters. Refusing to speak because
// the target cannot be read would leave the reader editing a release believing
// it is the source.
func TestOverrideMessageSpeaksThroughABrokenLink(t *testing.T) {
	root := t.TempDir()
	if err := os.Symlink(filepath.Join(root, "gone"), filepath.Join(root, ".madock")); err != nil {
		t.Fatalf("linking .madock: %v", err)
	}

	dir := filepath.Join(root, ".madock")
	joined := strings.Join(whereToEditLines(dir, filepath.Join(dir, "config.xml")), "\n")

	if !strings.Contains(joined, "repository") {
		t.Errorf("a broken link silenced the advice:\n%s", joined)
	}
}
