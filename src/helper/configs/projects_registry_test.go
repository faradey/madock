package configs

import (
	"os"
	"path/filepath"
	"testing"
)

// TestListProjectsIn covers the five things a registry entry can be, because four
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

	// And one pointing at nothing, which must be **reported**. It used to be
	// skipped, on the reasoning that an entry whose own directory is gone is not
	// an entry — true, and the wrong conclusion: the name is still in the
	// registry and nothing would say so. Measured on the BigCommerce cluster VM
	// on 2026-08-27, where four entries linked into a /tmp directory a reboot
	// had cleared: `project:list` answered "No projects are registered" and
	// `project:list --stale`, which exists for "the source is gone", answered
	// "Every registered project still has its source directory".
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
		"alive":       ProjectOk,
		"deleted":     ProjectMissingSource,
		"legacy":      ProjectNoPath,
		"linked":      ProjectOk,
		"broken-link": ProjectBrokenLink,
	}

	if len(got) != len(want) {
		t.Fatalf("entries = %v, want %v — a support directory is not an entry, and an entry that resolves to nothing still is one", got, want)
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

// The point of reporting a broken link rather than skipping it, stated as the
// filter that `project:list --stale` actually applies: anything that is not
// ProjectOk. Before the change the entry never reached this list, so the
// filtered form answered that everything was fine — which is the one wrong
// answer that reads as good news.
func TestStaleFilterSeesAnEntryThatResolvesToNothing(t *testing.T) {
	execDir := t.TempDir()

	if err := os.MkdirAll(filepath.Join(execDir, "projects"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(execDir, "gone"), filepath.Join(execDir, "projects", "cluster-run")); err != nil {
		t.Fatal(err)
	}

	stale := 0
	for _, entry := range ListProjectsIn(execDir) {
		if entry.State != ProjectOk {
			stale++
		}
	}

	if stale != 1 {
		t.Errorf("--stale would report %d entries, want 1 — an entry pointing at nothing is exactly what it is for", stale)
	}
}
