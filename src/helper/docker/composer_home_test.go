package docker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The cases here are about what is left on disk, not about a boolean, because
// the failure this guards is a layout: a project that believes it pins its own
// tool versions while running somebody else's.
//
// The one that must never regress is `isolate keeps the global home`: the
// project entry is a symlink, and removing a symlink must remove the link and
// not walk into it. Getting that wrong deletes the machine's composer home —
// every project's packages at once — while the code reads as if it tidied one
// project.

func lstatMode(t *testing.T, path string) os.FileMode {
	t.Helper()
	fi, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("lstat %s: %v", path, err)
	}
	return fi.Mode()
}

func newGlobalHome(t *testing.T) string {
	t.Helper()
	global := filepath.Join(t.TempDir(), "global-composer")
	if err := os.MkdirAll(filepath.Join(global, "cache"), 0o777); err != nil {
		t.Fatal(err)
	}
	// The two things the global home holds, so a test can tell them apart.
	if err := os.WriteFile(filepath.Join(global, "composer.json"), []byte(`{"require":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(global, "cache", "package.zip"), []byte("cached"), 0o644); err != nil {
		t.Fatal(err)
	}
	return global
}

func TestSharedHomeLinksToTheGlobalDirectory(t *testing.T) {
	global := newGlobalHome(t)
	project := filepath.Join(t.TempDir(), "composer")

	notice, err := prepareProjectComposerDir(project, global, true, true)
	if err != nil {
		t.Fatal(err)
	}
	if notice != "" {
		t.Errorf("a first link is routine and should say nothing, got %q", notice)
	}
	if lstatMode(t, project)&os.ModeSymlink == 0 {
		t.Fatal("the project entry is not a symlink")
	}
	target, _ := os.Readlink(project)
	if target != global {
		t.Errorf("link points at %q, want %q", target, global)
	}
}

func TestSharedHomeIsIdempotent(t *testing.T) {
	global := newGlobalHome(t)
	project := filepath.Join(t.TempDir(), "composer")

	if _, err := prepareProjectComposerDir(project, global, true, true); err != nil {
		t.Fatal(err)
	}
	notice, err := prepareProjectComposerDir(project, global, true, true)
	if err != nil {
		t.Fatalf("second run failed: %v", err)
	}
	if notice != "" {
		t.Errorf("nothing changed, so nothing should be said: %q", notice)
	}
}

// A real directory here is drift and is replaced — that is the long-standing
// invariant. What is new is that it stops being silent when something was in
// it, because "it reinstalled itself again" is otherwise unexplainable.
func TestSharedHomeSaysWhenItReplacesSomethingNonEmpty(t *testing.T) {
	global := newGlobalHome(t)
	project := filepath.Join(t.TempDir(), "composer")
	if err := os.MkdirAll(filepath.Join(project, "vendor"), 0o777); err != nil {
		t.Fatal(err)
	}

	notice, err := prepareProjectComposerDir(project, global, true, true)
	if err != nil {
		t.Fatal(err)
	}
	if notice == "" {
		t.Error("a non-empty directory was deleted without a word")
	}
	if !strings.Contains(notice, "shared_home=false") {
		t.Errorf("the notice does not say how to keep a per-project home: %q", notice)
	}
	if lstatMode(t, project)&os.ModeSymlink == 0 {
		t.Error("the entry should be a symlink after replacement")
	}
}

func TestIsolateReplacesTheLinkWithARealDirectory(t *testing.T) {
	global := newGlobalHome(t)
	project := filepath.Join(t.TempDir(), "composer")
	if _, err := prepareProjectComposerDir(project, global, true, true); err != nil {
		t.Fatal(err)
	}

	notice, err := prepareProjectComposerDir(project, global, false, true)
	if err != nil {
		t.Fatal(err)
	}
	if notice == "" {
		t.Error("switching to a private home reinstalls global tools — that has to be said")
	}
	if lstatMode(t, project)&os.ModeSymlink != 0 {
		t.Fatal("the project entry is still a symlink")
	}
	if fi, statErr := os.Stat(project); statErr != nil || !fi.IsDir() {
		t.Fatalf("expected a real directory at %s", project)
	}
}

// The whole point of the cheap half: downloads stay shared.
func TestIsolateKeepsTheCacheShared(t *testing.T) {
	global := newGlobalHome(t)
	project := filepath.Join(t.TempDir(), "composer")

	if _, err := prepareProjectComposerDir(project, global, false, true); err != nil {
		t.Fatal(err)
	}
	cache := filepath.Join(project, "cache")
	if lstatMode(t, cache)&os.ModeSymlink == 0 {
		t.Fatal("the cache is not shared")
	}
	body, err := os.ReadFile(filepath.Join(cache, "package.zip"))
	if err != nil || string(body) != "cached" {
		t.Errorf("the shared cache is not reachable through the link: %v %q", err, body)
	}
}

func TestIsolateWithPrivateCacheLinksNothing(t *testing.T) {
	global := newGlobalHome(t)
	project := filepath.Join(t.TempDir(), "composer")

	if _, err := prepareProjectComposerDir(project, global, false, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(project, "cache")); !os.IsNotExist(err) {
		t.Error("a private cache should be left for composer to create, not linked")
	}
}

// The one that would be catastrophic and is invisible in a diff: os.Remove on a
// symlink must remove the link. If it ever followed the link, the machine's
// composer home — every project's packages — would go with it.
func TestIsolateKeepsTheGlobalHomeIntact(t *testing.T) {
	global := newGlobalHome(t)
	project := filepath.Join(t.TempDir(), "composer")
	if _, err := prepareProjectComposerDir(project, global, true, true); err != nil {
		t.Fatal(err)
	}

	if _, err := prepareProjectComposerDir(project, global, false, true); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(global, "composer.json")); err != nil {
		t.Fatalf("the global composer.json is gone: %v", err)
	}
	if _, err := os.Stat(filepath.Join(global, "cache", "package.zip")); err != nil {
		t.Fatalf("the global cache is gone: %v", err)
	}
}

// Going back is a supported move — it is how somebody undoes the experiment.
func TestIsolateThenShareAgain(t *testing.T) {
	global := newGlobalHome(t)
	project := filepath.Join(t.TempDir(), "composer")

	if _, err := prepareProjectComposerDir(project, global, false, true); err != nil {
		t.Fatal(err)
	}
	if _, err := prepareProjectComposerDir(project, global, true, true); err != nil {
		t.Fatal(err)
	}
	if lstatMode(t, project)&os.ModeSymlink == 0 {
		t.Fatal("the project did not go back to the shared home")
	}
	if _, err := os.Stat(filepath.Join(global, "composer.json")); err != nil {
		t.Fatalf("the global home did not survive the round trip: %v", err)
	}
}
