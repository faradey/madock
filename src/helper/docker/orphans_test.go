package docker

import "testing"

// noScopes is a registry where no project has a scope but default.
func noScopes(string) []string { return nil }

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

	got := orphansOf("volume", lines, known, noScopes)

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

// Compose labels project ShopU madock_shopu. Compared with the registry as
// written, the live database volume of ShopU was listed as an orphan, with
// `docker volume rm` suggested for it. Reproduced on a VM before the fix.
func TestOrphansOfKnowsAProjectByComposesName(t *testing.T) {
	known := map[string]bool{"ShopU": true, "shop+1": true}

	got := orphansOf("volume", []string{
		"madock_shopu_dbdata\tmadock_shopu",
		"madock_shop1_dbdata\tmadock_shop1",
	}, known, noScopes)

	if len(got) != 0 {
		t.Errorf("a registered project's volumes were reported as orphans: %+v", got)
	}
}

// Project shop in scope staging runs as madock_shop-staging. Compared with the
// registry's names alone, that scope's live database was listed as the orphan
// of a project shop-staging, with `docker volume rm` suggested for it. A label
// with no such scope behind it is still an orphan.
func TestOrphansOfKnowsAProjectInItsScopes(t *testing.T) {
	known := map[string]bool{"Shop": true}
	scopes := func(name string) []string {
		if name == "Shop" {
			return []string{"Staging"}
		}
		return nil
	}

	got := orphansOf("volume", []string{
		"madock_shop-staging_dbdata\tmadock_shop-staging",
		"madock_shop-gone_dbdata\tmadock_shop-gone",
	}, known, scopes)

	if len(got) != 1 || got[0].Project != "shop-gone" {
		t.Errorf("want only shop-gone as an orphan, got %+v", got)
	}
}

// A registry entry madock cannot read is still an entry. It belongs to
// `project:list --stale`, which now reports it, and listing it here as well
// would ask two commands to argue about the same thing.
func TestOrphansOfLeavesRegisteredNamesAlone(t *testing.T) {
	known := map[string]bool{"broken-entry": true}

	got := orphansOf("network", []string{"madock_broken-entry_default\tmadock_broken-entry"}, known, noScopes)

	if len(got) != 0 {
		t.Errorf("a name the registry still holds was reported as an orphan: %+v", got)
	}
}
