package docker

import "testing"

// The rule this tests is the one that decides whether somebody is shown their
// own leftovers or a stranger's disk, so it is separated from the docker call
// and tested without a daemon.
func TestOrphansOf(t *testing.T) {
	known := map[string]bool{"live-project": true}

	lines := []string{
		// Belongs to a project that is still registered.
		"madock_live-project_dbdata\tmadock_live-project",
		// The case the command exists for: a volume that outlived its project.
		"madock_gone-project_dbdata\tmadock_gone-project",
		// Somebody else's compose stack on the same machine. Reporting it would
		// be calling a stranger's disk our mess.
		"someone_elses_data\tsomeone-elses-stack",
		// Malformed or unlabelled lines are not guesses to make.
		"no-tab-here",
		"\tmadock_gone-project",
		"orphan-with-no-label\t",
		// A label that is exactly the prefix names no project.
		"weird\tmadock_",
	}

	got := orphansOf("volume", lines, known)

	if len(got) != 1 {
		t.Fatalf("got %d orphans, want 1: %+v", len(got), got)
	}
	if got[0].Name != "madock_gone-project_dbdata" {
		t.Errorf("name = %q", got[0].Name)
	}
	// The name somebody would type, not the label docker stores.
	if got[0].Project != "gone-project" {
		t.Errorf("project = %q, want the prefix stripped", got[0].Project)
	}
	if got[0].Kind != "volume" {
		t.Errorf("kind = %q", got[0].Kind)
	}
}

// A registry entry madock cannot read is still an entry. It belongs to
// `project:list --stale`, which now reports it, and listing it here as well
// would ask two commands to argue about the same thing.
func TestOrphansOfLeavesRegisteredNamesAlone(t *testing.T) {
	known := map[string]bool{"broken-entry": true}

	got := orphansOf("network", []string{"madock_broken-entry_default\tmadock_broken-entry"}, known)

	if len(got) != 0 {
		t.Errorf("a name the registry still holds was reported as an orphan: %+v", got)
	}
}
