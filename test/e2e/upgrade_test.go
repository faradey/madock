//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// staleTemplate is a file every installation has and no project reads on a
// `custom` project, so editing it cannot change what the containers do — only
// whether the extraction noticed.
const staleTemplate = "docker/snippets/docker-compose/nginx.yml"

// TestAnOlderInstallationIsBroughtForward covers the upgrade, which is the one
// path every user takes and no test had.
//
// An installation is not only a registry: it is also the template tree unpacked
// beside the binary and a recorded version that decides which migrations still
// have to run. A new binary meeting an old installation has to fix both, and
// each of them fails silently in its own way — a stale template renders against
// a renderer that no longer understands it (which is how literal `{{{` reached
// the shared proxy and took every project on a machine down at once), and a
// version stamp that does not move means a migration never runs, so a renamed
// key is simply gone with nothing reported.
//
// The old installation is built rather than found: the version marker and the
// recorded version are the only two things that make an installation "old", and
// setting them is exactly what an older binary would have left behind.
func TestAnOlderInstallationIsBroughtForward(t *testing.T) {
	p := newProject(t, "e2eupgrade")

	p.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2eupgrade.test",
	)
	p.run(20*time.Minute, "start")

	// What this binary stamped, so the assertions below have something to
	// compare against rather than a version number written into the test.
	marker := filepath.Join(p.execDir, ".embedded_version")
	current := strings.TrimSpace(readFile(t, marker))
	if current == "" {
		t.Fatal("the installation carries no .embedded_version, so nothing here can tell an upgrade from a fresh install")
	}

	// Now make it look like an installation an older binary left behind: an old
	// stamp, an edited template, and a template that is missing altogether.
	template := filepath.Join(p.execDir, staleTemplate)
	original := readFile(t, template)
	writeString(t, template, original+"\n# left behind by an older release\n")

	withdrawn := filepath.Join(p.execDir, "docker", "snippets", "docker-compose", "varnish.yml")
	if err := os.Remove(withdrawn); err != nil {
		t.Fatalf("removing %s to simulate an incomplete tree: %v", withdrawn, err)
	}

	writeString(t, marker, "0.0.1")
	setRecordedVersion(t, p, "3.9.8")

	// Any command at all. The extraction and the migration both run before the
	// command does, which is the property being pinned: an upgrade is not a
	// step somebody remembers to take.
	p.run(3*time.Minute, "status")

	refreshed := readFile(t, template)
	if strings.Contains(refreshed, "left behind by an older release") {
		t.Error("an edited template survived the upgrade: the tree beside the binary is now older than the renderer that reads it")
	}
	requireFile(t, withdrawn, "a template missing from the tree must come back")

	if got := strings.TrimSpace(readFile(t, marker)); got != current {
		t.Errorf(".embedded_version is %q after the upgrade, want %q", got, current)
	}

	if got := recordedVersion(t, p); got == "3.9.8" || got == "" {
		t.Errorf("the recorded version is %q, so the migrations for everything after 3.9.8 will never run", got)
	}

	// And the project it was all done to is still a working project: the
	// database answers, and the generated stack was rewritten rather than left
	// as the old binary had it.
	p.run(25*time.Minute, "rebuild")
	p.freshTable("upgrade_probe", "(id INT)")
	p.query("INSERT INTO upgrade_probe VALUES (1)")
	requireContains(t, p.query("SELECT COUNT(*) FROM upgrade_probe"), "1", "the database after an upgrade and a rebuild")
}

// setRecordedVersion rewrites the version an installation believes it is on.
//
// Written into the file rather than set with `config:set`: `madock_version` is
// the installation's own bookkeeping, not a project setting, and an older
// binary is exactly what would have left an older number there.
func setRecordedVersion(t *testing.T, p *project, version string) {
	t.Helper()

	path := filepath.Join(p.execDir, "projects", "config.xml")
	body := readFile(t, path)

	const open, close = "<madock_version>", "</madock_version>"
	start := strings.Index(body, open)
	if start < 0 {
		t.Fatalf("no %s in %s — the installation records its version somewhere else now", open, path)
	}
	end := strings.Index(body[start:], close)
	if end < 0 {
		t.Fatalf("%s is not closed in %s", open, path)
	}

	writeString(t, path, body[:start+len(open)]+version+body[start+end:])
}

func recordedVersion(t *testing.T, p *project) string {
	t.Helper()

	body := readFile(t, filepath.Join(p.execDir, "projects", "config.xml"))
	const open, close = "<madock_version>", "</madock_version>"

	start := strings.Index(body, open)
	if start < 0 {
		return ""
	}
	start += len(open)
	end := strings.Index(body[start:], close)
	if end < 0 {
		return ""
	}
	return strings.TrimSpace(body[start : start+end])
}

func writeString(t *testing.T, path, body string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}
