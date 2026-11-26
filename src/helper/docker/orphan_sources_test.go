package docker

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// A project directory whose config.xml is gone — a removal stopped halfway —
// is listed. A registered project, an entry the registry holds though its
// config is missing, the support directories and plain files are not.
func TestAProjectDirectoryWithNoRegistryEntryIsListed(t *testing.T) {
	exec := t.TempDir()
	root := filepath.Join(exec, "projects")
	mk := func(rel string, withConfig bool) {
		t.Helper()
		dir := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Join(dir, "backup"), 0o755); err != nil {
			t.Fatal(err)
		}
		if withConfig {
			if err := os.WriteFile(filepath.Join(dir, "config.xml"), []byte("<config/>"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	mk("shop", true)
	// Set up after the registry was read: it has its config.xml, so it is an
	// entry, not litter, though known does not hold it yet.
	mk("fresh", true)
	mk("gone", false)
	mk("bin", false)
	mk("docker", false)
	mk("assets", false)
	mk("held", false)
	if err := os.WriteFile(filepath.Join(root, "config.xml"), []byte("<config/>"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := orphanProjectDirs(exec, map[string]bool{"shop": true, "held": true})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Project != "gone" || got[0].Kind != "directory" || got[0].Name != filepath.Join(root, "gone") {
		t.Errorf("listed %+v, want only projects/gone", got)
	}
}

// No projects/ at all is an installation with no projects, not an error.
func TestNoProjectsDirectoryIsNothingNotUnknown(t *testing.T) {
	got, err := orphanProjectDirs(t.TempDir(), nil)
	if err != nil || len(got) != 0 {
		t.Errorf("got %+v, %v", got, err)
	}
}

// A source that fails is named, never folded into "nothing found", and what
// another source listed is kept.
func TestAFailingSourceIsUnknownNotNothing(t *testing.T) {
	sources := []namedSource{
		{name: "zfs datasets", find: func(map[string]bool) ([]Orphan, error) {
			return nil, errors.New("zfs: command not found")
		}},
		{name: "other", find: func(map[string]bool) ([]Orphan, error) {
			return []Orphan{{Kind: "dataset", Name: "tank/madock/gone", Project: "gone"}}, nil
		}},
	}
	found, unknown := fromSources(sources, nil)
	if len(unknown) != 1 || unknown[0].Source != "zfs datasets" || unknown[0].Error == "" {
		t.Errorf("unknown = %+v, want the zfs source with its error", unknown)
	}
	if len(found) != 1 || found[0].Name != "tank/madock/gone" {
		t.Errorf("found = %+v, want the other source's dataset", found)
	}
}
