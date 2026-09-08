//go:build e2e

package e2e

import (
	"os"
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

// TestAProjectShippedWithoutNginxNeverTakesTheWebPorts is the other order, and
// it is the one real projects use.
//
// `nginx/enabled` is not usually typed at a machine: it is committed in the
// project's own `.madock/config.xml`, which is how extmag-core-shopify ships —
// Core accepts no request from anywhere, so a vhost would mean a certificate, a
// block in the shared proxy and an open port for nothing. That file is read
// before the first render, so the switch is off the first time anything asks
// for a port.
//
// Which is what makes the claim in `configs.NginxEnabled` — "false leaves out
// the container, the vhost, the block in the shared proxy, the name in the
// certificate and the ports" — true of this path and false of the other. A
// project set up with the web server on and switched off afterwards keeps the
// numbers it already has: `setup` renders, and allocates, before there is a
// project for `config:set` to write to, and nothing hands a number back except
// `project:remove`. Measured both ways in the VM on 2026-09-08.
func TestAProjectShippedWithoutNginxNeverTakesTheWebPorts(t *testing.T) {
	install := newInstallation(t)
	p := install.project("e2eshippedoff")

	// Written before setup, exactly as a repository carries it.
	madockDir := filepath.Join(p.runDir, ".madock")
	if err := os.MkdirAll(madockDir, 0o755); err != nil {
		t.Fatalf("creating %s: %v", madockDir, err)
	}
	// The platform and the language belong in this file as well, exactly as
	// extmag-core-shopify carries them. A file holding only the nginx switch
	// looks like it should work and does not: the project's own config wins
	// over what `setup` was told, so app.Dockerfile rendered from a config with
	// no platform and docker stopped with "failed to read dockerfile:
	// app.Dockerfile". Measured, and it is the reason this fixture is not
	// shorter.
	writeString(t, filepath.Join(madockDir, "config.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<config>
    <scopes>
        <default>
            <platform>custom</platform>
            <language>none</language>
            <nginx>
                <enabled>false</enabled>
            </nginx>
        </default>
    </scopes>
</config>
`)

	p.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2eshippedoff.test",
	)
	p.run(20*time.Minute, "start")

	compose := readFile(t, p.generated("docker-compose.yml"))
	if strings.Contains(compose, "\n  nginx:") {
		t.Fatalf("the project declares an nginx service, so its own config.xml was not read and nothing below means anything:\n%s", compose)
	}

	registry := readFile(t, filepath.Join(install.dir, "aruntime", "ports.conf"))
	for _, key := range []string{p.name + "/nginx=", p.name + "/nginx_ssl="} {
		if strings.Contains(registry, key) {
			t.Errorf("%s is reserved for a project that ships without a web server:\n%s", key, registry)
		}
	}
}
