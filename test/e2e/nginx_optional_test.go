//go:build e2e

package e2e

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestNginxOffKeepsTheProjectOutOfTheSharedProxy is the end-to-end half of
// `nginx/enabled`, and it is the half the unit tests cannot reach.
//
// Those decide what one project renders. This decides what happens to the file
// every project on the machine is served through: a project with no web server
// must lose its server block, its neighbours must keep theirs, and the next
// start of any other project must not put it back. Routing a hostname at a
// container that does not exist shows up as a 502 that nobody configured.
//
// Two projects, because one cannot show the second half: the block is rebuilt
// on somebody else's `start`, not on the switch.
//
// **What the sabotage measured, and it is worth writing down.** The generator
// both deletes the project's cached block and skips the project; deleting the
// cache is what the code comments are about, and leaving that deletion out
// changes nothing observable — this test passed against a binary that skipped
// the cache instead of removing it, because the skip happens *before* anything
// reads the cache. What the test does discriminate is the skip itself: with
// that removed, both assertions go red, the routing and the resurrection alike.
// So the cache deletion is defence against a future reordering rather than
// something any test can currently see, and this comment is the only place that
// says so.
func TestNginxOffKeepsTheProjectOutOfTheSharedProxy(t *testing.T) {
	install := newInstallation(t)

	quiet := install.project("e2enonginx")
	neighbour := install.project("e2ewithnginx")

	for _, p := range []*project{quiet, neighbour} {
		p.run(5*time.Minute, "setup", "-y",
			"--platform=custom",
			"--language=none",
			"--hosts="+p.name+".test",
		)
	}

	// The starting state is asserted rather than assumed. A project that was
	// never routed would satisfy every assertion below without the switch doing
	// anything — the same shape of mistake as testing `health` against a value
	// that was already there.
	quiet.run(20*time.Minute, "start")
	neighbour.run(20*time.Minute, "start")

	proxyConf := filepath.Join(install.dir, "aruntime", "ctx", "proxy.conf")
	requireContains(t, readFile(t, proxyConf), "e2enonginx.test", "the project's routing before the web server is switched off")

	quiet.run(2*time.Minute, "config:set", "-n", "nginx/enabled", "-v", "false")
	quiet.run(25*time.Minute, "rebuild")

	// The compose file first: no web server container at all.
	compose := readFile(t, quiet.generated("docker-compose.yml"))
	if strings.Contains(compose, "\n  nginx:") {
		t.Errorf("the project still declares an nginx service after nginx/enabled=false:\n%s", compose)
	}

	afterOff := readFile(t, proxyConf)
	if strings.Contains(afterOff, "e2enonginx.test") {
		t.Errorf("a project with no web server is still routed by the shared proxy:\n%s", afterOff)
	}
	requireContains(t, afterOff, "e2ewithnginx.test", "the neighbour's routing must survive the rewrite")

	// The resurrection. `start` regenerates the shared configuration from every
	// project's cached block, so a cache that was skipped rather than deleted
	// comes back here and nowhere else.
	neighbour.run(20*time.Minute, "start")

	afterNeighbour := readFile(t, proxyConf)
	if strings.Contains(afterNeighbour, "e2enonginx.test") {
		t.Errorf("the neighbour's start brought back the routing of a project with no web server:\n%s", afterNeighbour)
	}

	// **The ports are deliberately not asserted here, and the measurement is
	// why.** `nginx/enabled` documents itself as leaving out "the container,
	// the vhost, the block in the shared proxy, the name in the certificate and
	// the ports", and the first two versions of this test tried to pin the last
	// of those. Both were wrong, in different ways, and the second one is the
	// interesting one:
	//
	//   - matching the word "nginx" in the registry hits every line of a project
	//     called e2enonginx, so it reported ten reserved web servers where there
	//     were none;
	//   - and a project set up with the switch already off still holds
	//     `<name>/nginx` and `<name>/nginx_ssl`, because `setup` renders the
	//     project — and allocates — before there is any project to run
	//     `config:set` against. Measured in the VM: e2enoweb held both after a
	//     setup, a config:set and a start.
	//
	// So the claim is true of what the switch decides from then on and false of
	// what a project already holds, and nothing releases the numbers. Asserting
	// it here would pin a behaviour that does not exist; it is written up in
	// MADOCK-E2E-PLAN.md instead, where it can be decided rather than enforced
	// by a red test.
}
