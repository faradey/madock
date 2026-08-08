//go:build e2e

package e2e

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestRemovingAProjectTakesItOutOfTheProxy pins where routing is allowed to
// change, which is a decision and not an implementation detail.
//
// A stopped project keeps its server block. It is coming back, and rewriting the
// shared configuration — the one every other project is served through — for a
// pause would be churn and risk in exchange for nothing.
//
// A removed project is not coming back, and its block points at a container that
// no longer exists. Nothing used to take it out, so it survived until something
// else happened to regenerate the file.
//
// The other half is blast radius: removal rewrites the configuration every other
// project is served through, so this checks the neighbour is still served
// afterwards — in the file and over TLS, which are not the same claim.
func TestRemovingAProjectTakesItOutOfTheProxy(t *testing.T) {
	install := newInstallation(t)

	staying := install.project("e2estays")
	leaving := install.project("e2eleaves")

	for _, p := range []*project{staying, leaving} {
		p.run(5*time.Minute, "setup", "-y",
			"--platform=custom",
			"--language=none",
			"--hosts="+p.name+".test",
		)
		p.run(20*time.Minute, "start")
	}

	proxyConf := filepath.Join(install.dir, "aruntime", "ctx", "proxy.conf")
	both := readFile(t, proxyConf)
	requireContains(t, both, "e2estays.test", "the routing of the project that stays")
	requireContains(t, both, "e2eleaves.test", "the routing of the project that leaves")

	// Stopping changes nothing about routing, deliberately.
	leaving.run(5*time.Minute, "stop")

	afterStop := readFile(t, proxyConf)
	requireContains(t, afterStop, "e2eleaves.test", "a stopped project keeps its routing")
	requireContains(t, afterStop, "e2estays.test", "and so does its neighbour")

	// Removing does.
	leaving.run(5*time.Minute, "project:remove", "--force", "--name=e2eleaves")

	afterRemove := readFile(t, proxyConf)
	if strings.Contains(afterRemove, "e2eleaves.test") {
		t.Errorf("the removed project is still routed:\n%s", afterRemove)
	}

	requireContains(t, afterRemove, "e2estays.test", "the neighbour's routing survived the removal")
	requireCertificateFor(t, "e2estays.test")
}
