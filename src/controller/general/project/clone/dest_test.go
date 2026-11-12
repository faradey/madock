package clone

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Without --path nothing changes: beside the source, named after the clone.
func TestWithoutPathTheCloneGoesBesideTheSource(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "shop")
	if err := os.Mkdir(src, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := resolveTarget(src, "", "shop-try", "/nowhere", "/home/x")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(root, "shop-try"); got.Dir != want || got.Created != want {
		t.Errorf("got %+v, want Dir and Created %s", got, want)
	}
}

// Test runs clone a project many times; the clones belong in a scratch tree,
// not next to a repository checkout. A relative --path is the caller's
// directory's, and every missing directory on the way is the run's to create
// — and so the run's to remove when the clone fails.
func TestPathIsRelativeToTheCallerAndCreatesWhatIsMissing(t *testing.T) {
	root := t.TempDir()
	// The caller is not in the source's parent: a relative path taken against
	// the source would land somewhere else.
	src := filepath.Join(root, "repos", "shop")
	cwd := filepath.Join(root, "temp")
	for _, d := range []string{src, cwd} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	got, err := resolveTarget(src, "runs/r1", "shop-it1", cwd, "/home/x")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(root, "temp", "runs", "r1", "shop-it1"); got.Dir != want {
		t.Errorf("Dir %s, want %s", got.Dir, want)
	}
	if want := filepath.Join(root, "temp", "runs"); got.Created != want {
		t.Errorf("Created %s, want the first missing directory %s", got.Created, want)
	}
}

func TestPathAcceptsAbsoluteAndHome(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "shop")

	abs, err := resolveTarget(src, filepath.Join(root, "elsewhere"), "c1", "/cwd", "/home/x")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(root, "elsewhere", "c1"); abs.Dir != want {
		t.Errorf("absolute: %s, want %s", abs.Dir, want)
	}

	home, err := resolveTarget(src, "~/Projects/temp", "c1", "/cwd", root)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(root, "Projects", "temp", "c1"); home.Dir != want {
		t.Errorf("home: %s, want %s", home.Dir, want)
	}

	if _, err := resolveTarget(src, "~bob/x", "c1", "/cwd", root); err == nil {
		t.Error("~bob/ was accepted as if it were a path")
	}
}

// The clone unpacks its code with `rm -rf /var/www/html/*` over the mount: a
// target with something in it would lose it. An empty directory is fine, and
// stays when the clone fails — it was not this run's.
func TestPathRefusesATargetWithSomethingInIt(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "shop")

	full := filepath.Join(root, "dest", "c1")
	if err := os.MkdirAll(full, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(full, "keep.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := resolveTarget(src, filepath.Join(root, "dest"), "c1", "/cwd", "/home/x"); err == nil || !strings.Contains(err.Error(), "not empty") {
		t.Errorf("a non-empty target was accepted: %v", err)
	}

	file := filepath.Join(root, "dest", "c2")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := resolveTarget(src, filepath.Join(root, "dest"), "c2", "/cwd", "/home/x"); err == nil {
		t.Error("a file in the clone's place was accepted")
	}

	empty := filepath.Join(root, "dest", "c3")
	if err := os.Mkdir(empty, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := resolveTarget(src, filepath.Join(root, "dest"), "c3", "/cwd", "/home/x")
	if err != nil {
		t.Fatalf("an empty directory was refused: %v", err)
	}
	if got.Created != "" {
		t.Errorf("an existing directory would be removed on failure: Created %q", got.Created)
	}
}

// Inside the source the clone would ride along in the source's code mount,
// and into the source's next clone and snapshot.
func TestPathRefusesATargetInsideTheSource(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "shop")
	if _, err := resolveTarget(src, "var/clones", "c1", src, "/home/x"); err == nil || !strings.Contains(err.Error(), "inside the source") {
		t.Errorf("a target inside the source was accepted: %v", err)
	}
	// A sibling whose name only starts like the source is not inside it.
	if _, err := resolveTarget(src, filepath.Join(root, "shop-clones"), "c1", "/cwd", "/home/x"); err != nil {
		t.Errorf("a sibling directory was refused: %v", err)
	}
}
