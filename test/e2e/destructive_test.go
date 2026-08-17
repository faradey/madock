//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestDestructiveCommandsObeyTheInstallation covers the guard that stops
// `project:remove` and `prune` on an installation that has said they may not
// run — and, more importantly, covers the half that makes it worth having:
// that a project cannot lift it.
//
// The unit tests pin the resolver. This one pins the binary, because the two
// can disagree in exactly one way that matters. Every other setting in
// config.xml is reachable from a project — its own file overrides it and
// `config:set` writes it — so a key that ended up under <scopes> would look
// identical in review and be switchable by the very thing it protects against.
// Here the attempt is actually made, through the real command, and the refusal
// has to survive it.
//
// Nothing is started: the guard runs before the command touches Docker, and a
// test that started a project would pay twenty minutes to prove something the
// refusal makes unnecessary.
func TestDestructiveCommandsObeyTheInstallation(t *testing.T) {
	install := newInstallation(t)
	p := install.project("e2eguarded")

	p.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2eguarded.test",
	)

	registry := filepath.Join(p.execDir, "projects", "e2eguarded")

	// Restored before the harness cleans up, which removes the project the way
	// every other test does. Cleanups run last-registered-first, so this one is
	// ahead of the harness's, which was registered when the project was made.
	t.Cleanup(func() { writeInstallationConfig(t, p.execDir, "true") })
	writeInstallationConfig(t, p.execDir, "false")

	for _, refused := range [][]string{
		{"project:remove", "--force", "--name=e2eguarded"},
		{"prune"},
	} {
		out, err := p.tryRun(2*time.Minute, refused...)
		if err == nil {
			t.Errorf("`%s` ran on an installation that forbids it:\n%s", strings.Join(refused, " "), out)
		}
		// The refusal is only useful if it says what to edit. Somebody meets
		// this message on a server, in a hurry, and the next question is always
		// the same one.
		requireContains(t, out, "allow_destructive_commands", "the refusal should name the setting")
		requireContains(t, out, filepath.Join(p.execDir, "config.xml"), "the refusal should name the file to edit")
	}

	if _, err := os.Stat(registry); err != nil {
		t.Fatalf("the project was removed despite the refusal: %v", err)
	}

	// Now try to lift it, by every route a project has.
	//
	// The first is the strongest result available: `config:set` does not know
	// the key at all, because it only writes what is in the project's own
	// option set and this key is never in it. An error here is the assertion,
	// not a problem with the test.
	out, err := p.tryRun(2*time.Minute, "config:set", "-n", "allow_destructive_commands", "-v", "true")
	if err == nil {
		t.Errorf("config:set accepted a key that belongs to the installation:\n%s", out)
	}
	requireContains(t, out, "doesn't exist", "config:set should say the option is not a project option")

	// The other two are hand-edited files, which is what somebody does when a
	// command refuses. Both sit in the inheritance chain and neither is read
	// for this key.
	claimTheKey(t, filepath.Join(p.execDir, "projects", "e2eguarded", "config.xml"))
	claimTheKey(t, projectLocalConfig(t, p))

	out, err = p.tryRun(2*time.Minute, "project:remove", "--force", "--name=e2eguarded")
	if err == nil {
		t.Fatalf("a project turned the guard off for itself:\n%s", out)
	}
	if _, err := os.Stat(registry); err != nil {
		t.Fatalf("the project was removed after a project-level override: %v", err)
	}

	// And the installation's own file still has the last word, or turning the
	// guard back on would mean installing a different binary.
	writeInstallationConfig(t, p.execDir, "true")

	install.destroyProxy(p)
	p.run(5*time.Minute, "project:remove", "--force", "--name=e2eguarded")

	if _, err := os.Stat(registry); !os.IsNotExist(err) {
		t.Errorf("the project survived a removal the installation allows: %s", registry)
	}
}

// writeInstallationConfig writes {execDir}/config.xml carrying one answer.
//
// Written whole rather than edited: madock does not create this file on a fresh
// install, so a test that edited it would only ever run against a state that
// does not exist on a new machine.
func writeInstallationConfig(t *testing.T, execDir, allow string) {
	t.Helper()

	body := `<?xml version="1.0" encoding="UTF-8"?>
<config>
    <allow_destructive_commands>` + allow + `</allow_destructive_commands>
</config>
`
	if err := os.WriteFile(filepath.Join(execDir, "config.xml"), []byte(body), 0o644); err != nil {
		t.Fatalf("writing the installation config: %v", err)
	}
}

// projectLocalConfig creates the project's own .madock/config.xml the way the
// documentation tells a user to: a copy of the registry's config, in the
// project directory, under version control.
//
// `setup` does not create it — it is opt-in — so a test that only edited an
// existing file would silently skip this route entirely.
func projectLocalConfig(t *testing.T, p *project) string {
	t.Helper()

	body, err := os.ReadFile(filepath.Join(p.execDir, "projects", p.name, "config.xml"))
	if err != nil {
		t.Fatalf("reading the registry config: %v", err)
	}

	dir := filepath.Join(p.runDir, ".madock")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("creating %s: %v", dir, err)
	}

	path := filepath.Join(dir, "config.xml")
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}

	return path
}

// claimTheKey writes the guard's key into a config file, in both places a
// person would plausibly put it: at the top level, and inside the scope where
// every other setting lives.
func claimTheKey(t *testing.T, path string) {
	t.Helper()

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}

	claim := "<allow_destructive_commands>true</allow_destructive_commands>"
	edited := strings.Replace(string(body), "<config>", "<config>\n    "+claim, 1)
	edited = strings.Replace(edited, "<default>", "<default>\n            "+claim, 1)

	if edited == string(body) {
		t.Fatalf("%s had neither <config> nor <default> to write into:\n%s", path, body)
	}
	if err := os.WriteFile(path, []byte(edited), 0o644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}
