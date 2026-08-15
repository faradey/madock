package configs

import (
	"os"
	"path/filepath"
	"testing"
)

// TestListProjectsIn covers the four things a registry entry can be, because three
// of them were invisible before this existed.
//
// On the machine this was written on, two of fifty-eight entries pointed at
// directories that had been deleted — and both still held their port reservations
// and still had a server block in the generated proxy configuration, routing their
// hosts at containers that cannot exist. Nothing reported it.
//
// The fourth case is the one that trips a listing written against directory names:
// aruntime/projects/ holds `composer` and `ssh` beside the projects, and a directory
// without a config.xml is not an entry at all.
func TestListProjectsIn(t *testing.T) {
	execDir := t.TempDir()

	write := func(name, body string) {
		t.Helper()
		dir := filepath.Join(execDir, "projects", name)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("creating %s: %v", dir, err)
		}
		if body == "" {
			return
		}
		if err := os.WriteFile(filepath.Join(dir, "config.xml"), []byte(body), 0644); err != nil {
			t.Fatalf("writing config for %s: %v", name, err)
		}
	}

	config := func(path string) string {
		return `<?xml version="1.0" encoding="UTF-8"?>
<config>
    <scopes>
        <default>
            <path>` + path + `</path>
        </default>
    </scopes>
</config>`
	}

	live := t.TempDir()
	write("alive", config(live))
	write("deleted", config(filepath.Join(execDir, "not-here")))

	// A registry entry that is a symlink to its project, which is how every
	// installation set up from a temporary checkout looks. os.ReadDir answers about
	// the entry rather than its target, so this case was dropped before anything
	// looked at its configuration: on a cluster VM with four such projects running,
	// project:list said "No projects are registered". The first version of this test
	// built only real directories, which is exactly why it passed.
	linked := t.TempDir()
	linkedRegistry := filepath.Join(t.TempDir(), "linked-entry")
	if err := os.MkdirAll(linkedRegistry, 0755); err != nil {
		t.Fatalf("creating the symlink target: %v", err)
	}
	if err := os.WriteFile(filepath.Join(linkedRegistry, "config.xml"), []byte(config(linked)), 0644); err != nil {
		t.Fatalf("writing the linked config: %v", err)
	}
	if err := os.Symlink(linkedRegistry, filepath.Join(execDir, "projects", "linked")); err != nil {
		t.Fatalf("linking the registry entry: %v", err)
	}

	// And one pointing at nothing, which must be skipped rather than reported: an
	// entry whose own directory is gone is not an entry.
	if err := os.Symlink(filepath.Join(execDir, "gone"), filepath.Join(execDir, "projects", "broken-link")); err != nil {
		t.Fatalf("linking the broken registry entry: %v", err)
	}
	write("legacy", `<?xml version="1.0" encoding="UTF-8"?>
<config>
    <scopes>
        <default>
            <platform>magento2</platform>
        </default>
    </scopes>
</config>`)
	write("composer", "") // a support directory, not a project

	got := map[string]string{}
	for _, entry := range ListProjectsIn(execDir) {
		got[entry.Name] = entry.State
	}

	want := map[string]string{
		"alive":   ProjectOk,
		"deleted": ProjectMissingSource,
		"legacy":  ProjectNoPath,
		"linked":  ProjectOk,
	}

	if len(got) != len(want) {
		t.Fatalf("entries = %v, want %v — a directory without a config.xml is not a project", got, want)
	}
	for name, state := range want {
		if got[name] != state {
			t.Errorf("%s = %q, want %q", name, got[name], state)
		}
	}
}

// TestListProjectsInWithoutRegistry: an installation that has never set up a
// project has no projects directory, and asking about it is not an error.
func TestListProjectsInWithoutRegistry(t *testing.T) {
	if entries := ListProjectsIn(t.TempDir()); entries != nil {
		t.Errorf("entries = %v, want none", entries)
	}
}
