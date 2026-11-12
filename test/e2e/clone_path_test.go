//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestCloneGoesWhereThePathSays — `project:clone --path` puts the clone in a
// directory of the caller's choosing instead of beside the source.
//
// Test runs clone a project dozens of times, and beside the source meant beside
// a repository checkout: the throwaway clones of one integration suite piled up
// next to its sources. The path is the caller's, so a relative one is taken
// from where the command runs, and the directories on the way are created.
//
// A target that already holds something is refused before anything is made:
// the clone unpacks its code with `rm -rf /var/www/html/*` over that directory.
func TestCloneGoesWhereThePathSays(t *testing.T) {
	install := newInstallation(t)
	source := install.project("e2epathsrc")

	source.run(setupTimeout, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2epathsrc.test",
	)
	source.run(20*time.Minute, "start")
	source.freshTable("shared", "(note VARCHAR(32))")
	source.query("INSERT INTO shared VALUES ('written-in-source')")

	// Relative to the source's directory, where the command runs; runs/ and
	// runs/r1/ do not exist yet.
	want := filepath.Join(install.root, "runs", "r1", "e2epathcopy")
	out := source.run(30*time.Minute, "project:clone", "-n", "e2epathcopy", "-s=-copy",
		"--path=../runs/r1", "--json")
	clone := install.projectAt("e2epathcopy", want)

	requireContains(t, out, want, "--json should name the directory the clone was put in")
	if info, err := os.Stat(want); err != nil || !info.IsDir() {
		t.Fatalf("the clone's directory is not at %s: %v", want, err)
	}
	if _, err := os.Stat(filepath.Join(install.root, "e2epathcopy")); err == nil {
		t.Errorf("a directory for the clone was made beside the source as well")
	}

	clone.run(20*time.Minute, "start")
	copied := clone.query("SELECT note FROM shared")
	requireContains(t, copied, "written-in-source", "the clone at --path should start as a copy of the source")

	// Something already there: refused, untouched, nothing registered.
	full := filepath.Join(install.root, "full", "e2epathtaken")
	if err := os.MkdirAll(full, 0o755); err != nil {
		t.Fatal(err)
	}
	keep := filepath.Join(full, "keep.txt")
	if err := os.WriteFile(keep, []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	refused, err := source.tryRun(5*time.Minute, "project:clone", "-n", "e2epathtaken", "-s=-taken",
		"--path="+filepath.Join(install.root, "full"))
	if err == nil {
		t.Fatalf("a clone into a directory with a file in it was made:\n%s", refused)
	}
	requireContains(t, refused, "not empty", "the refusal should say why")
	if data, err := os.ReadFile(keep); err != nil || string(data) != "mine" {
		t.Errorf("the refused clone touched what was in its target: %q, %v", data, err)
	}
	if _, err := os.Stat(filepath.Join(install.dir, "projects", "e2epathtaken")); err == nil {
		t.Errorf("the refused clone was registered in the installation")
	}
}
