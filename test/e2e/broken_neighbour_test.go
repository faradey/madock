//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestABrokenProjectConfigDoesNotBlockItsNeighbours pins the decision that one
// project's unparsable config.xml is that project's problem, not the machine's.
//
// The shared proxy.conf builder reads every project's config on every start, and
// GetProjectConfig's Fatalln turned one broken file into a start, clone or stop
// that failed for every other project. Measured on a live machine: a bad config
// in one project broke `nginx -t` of the shared proxy and every project lost
// routing.
//
// So this starts two projects, corrupts one's config, and asserts the other
// still starts and is still routed — and that the broken project itself still
// stops, instead of exiting 1 on the parse before any docker call.
func TestABrokenProjectConfigDoesNotBlockItsNeighbours(t *testing.T) {
	install := newInstallation(t)

	good := install.project("e2egood")
	broken := install.project("e2ebroken")

	for _, p := range []*project{good, broken} {
		p.run(setupTimeout, "setup", "-y",
			"--platform=custom",
			"--language=none",
			"--hosts="+p.name+".test",
		)
		p.run(20*time.Minute, "start")
	}

	// Corrupt the broken project's installation config.xml — the first file the
	// proxy builder reads for it, and the one GetProjectConfig would Fatalln on.
	// Its exact contents are put back before this test returns so the harness's
	// own teardown, which runs project:remove on it, is not asked to remove a
	// project whose config does not parse: project:remove reads that config
	// directly and tolerating it there is a separate change.
	brokenConf := filepath.Join(install.dir, "projects", "e2ebroken", "config.xml")
	original, err := os.ReadFile(brokenConf)
	if err != nil {
		t.Fatal(err)
	}
	defer os.WriteFile(brokenConf, original, 0o644)
	if err := os.WriteFile(brokenConf, []byte("<config><scopes><default>not closed"), 0o644); err != nil {
		t.Fatal(err)
	}

	// The good project still starts and is still routed, though the render reads
	// the broken neighbour's config on the way.
	good.run(20*time.Minute, "start")

	proxyConf := filepath.Join(install.dir, "aruntime", "ctx", "proxy.conf")
	after := readFile(t, proxyConf)
	requireContains(t, after, "e2egood.test", "the good project is routed despite a neighbour's broken config")

	// And stopping the broken project itself still stops it, rather than exiting
	// 1 on the parse before any docker call.
	broken.run(5*time.Minute, "stop")
}

// TestABrokenOwnConfigDoesNotBlockItsNeighbours is the same decision for the
// file a person actually edits: the project's own .madock/config.xml, in its
// source directory. The test above breaks only the installation's runtime copy,
// and the check it relies on never reached this file — it read an unscoped
// `path` the runtime file does not hold — so a broken own file still took down
// every neighbour's start and clone and the project's own stop, while that test
// stayed green.
//
// An empty root, `<config/>`, is covered beside it: the decoder hands it back as
// a string, and asserting it a map panicked, which no skip could catch. It is a
// config that sets nothing, and its project is read, not skipped.
func TestABrokenOwnConfigDoesNotBlockItsNeighbours(t *testing.T) {
	install := newInstallation(t)

	good := install.project("e2eowngood")
	broken := install.project("e2eownbroken")
	empty := install.project("e2eownempty")

	for _, p := range []*project{good, broken, empty} {
		p.run(setupTimeout, "setup", "-y",
			"--platform=custom",
			"--language=none",
			"--hosts="+p.name+".test",
		)
		p.run(20*time.Minute, "start")
	}

	// Each own file is put back before the test returns, so the harness's
	// teardown removes ordinary projects.
	replaceOwn := func(p *project, body string) {
		t.Helper()
		own := filepath.Join(p.runDir, ".madock", "config.xml")
		original, err := os.ReadFile(own)
		if err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if original == nil {
				os.Remove(own)
				return
			}
			os.WriteFile(own, original, 0o644)
		})
		if err := os.MkdirAll(filepath.Dir(own), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(own, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	replaceOwn(broken, "<config><scopes><default>not closed")
	replaceOwn(empty, "<config/>")

	// The neighbour starts, the broken project is skipped and named by its own
	// file, and the empty one is not skipped at all.
	out := good.run(20*time.Minute, "start")
	requireContains(t, out, `Skipping project "e2eownbroken"`, "the broken project is skipped by name")
	requireContains(t, out, filepath.Join("e2eownbroken", ".madock", "config.xml"), "the message names the file that does not parse")
	if strings.Contains(out, `Skipping project "e2eownempty"`) {
		t.Errorf("a project whose own config sets nothing was skipped as broken:\n%s", out)
	}
	proxyConf := filepath.Join(install.dir, "aruntime", "ctx", "proxy.conf")
	requireContains(t, readFile(t, proxyConf), "e2eowngood.test", "the good project is routed despite the broken neighbour")

	// A clone renders the same shared proxy, so it reads the same neighbours.
	good.run(30*time.Minute, "project:clone", "-n", "e2eownclone", "-s=-copy")
	install.project("e2eownclone")

	// The broken project still stops; the empty one stops too.
	broken.run(5*time.Minute, "stop")
	empty.run(5*time.Minute, "stop")
}
