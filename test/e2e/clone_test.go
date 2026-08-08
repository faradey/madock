//go:build e2e

package e2e

import (
	"strings"
	"testing"
	"time"
)

// TestCloneCopiesTheDataAndThenDiverges covers what a clone is for: a second
// copy of a working project to try something dangerous on.
//
// That makes two properties matter, and they pull in opposite directions. The
// clone has to start as a real copy — an empty database would make it useless
// for reproducing anything. And from that moment the two must not share
// anything, because the whole point is that the copy can be broken safely. A
// clone that writes into the original's database is the worst outcome madock
// has: the person doing it believes they are working on a throwaway.
//
// The hosts have to diverge too. Both projects are behind one proxy, and two
// projects claiming the same name is a coin toss over which site answers.
func TestCloneCopiesTheDataAndThenDiverges(t *testing.T) {
	install := newInstallation(t)
	source := install.project("e2eclone")

	source.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2eclone.test",
	)
	source.run(20*time.Minute, "start")

	source.freshTable("shared", "(note VARCHAR(32))")
	source.query("INSERT INTO shared VALUES ('written-in-source')")

	// The suffix goes in before the TLD dot: e2eclone.test becomes
	// e2eclone-copy.test. It is required precisely because two projects on one
	// proxy cannot both answer to the same name.
	source.run(30*time.Minute, "project:clone", "-n", "e2eclonecopy", "-s", "-copy")

	// The clone lands beside the source, which is where the harness would have
	// put a project of that name anyway. Registering it now gives it the same
	// cleanup as everything else.
	clone := install.project("e2eclonecopy")
	clone.run(20*time.Minute, "start")

	hosts := clone.run(2*time.Minute, "config:list")
	requireContains(t, hosts, "e2eclone-copy.test", "the clone's host should carry the suffix")
	if strings.Contains(hosts, "name e2eclone.test") {
		t.Errorf("the clone still answers to the source's host:\n%s", hosts)
	}

	copied := clone.query("SELECT note FROM shared")
	requireContains(t, copied, "written-in-source", "the clone should start as a copy of the source")

	// From here they are separate databases, and each has to prove it does not
	// see the other's writes.
	clone.query("INSERT INTO shared VALUES ('written-in-clone')")
	source.query("INSERT INTO shared VALUES ('written-in-source-after-clone')")

	inSource := source.query("SELECT note FROM shared")
	if strings.Contains(inSource, "written-in-clone") {
		t.Errorf("a write in the clone reached the source database:\n%s", inSource)
	}

	inClone := clone.query("SELECT note FROM shared")
	if strings.Contains(inClone, "written-in-source-after-clone") {
		t.Errorf("a write in the source reached the clone's database:\n%s", inClone)
	}
}
