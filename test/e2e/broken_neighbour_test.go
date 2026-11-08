//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
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
	brokenConf := filepath.Join(install.dir, "projects", "e2ebroken", "config.xml")
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
