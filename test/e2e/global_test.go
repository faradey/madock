//go:build e2e

package e2e

import (
	"strings"
	"testing"
	"time"
)

// TestGlobalServiceReachesTheNextProject covers the flag with the widest blast
// radius in the whole command set.
//
// `service:enable --global` does not change one project. It writes into the
// installation's own configuration, so it decides what every project created
// afterwards is made of. Getting that wrong is quiet in both directions: a
// --global that only wrote locally leaves the person wondering why their new
// projects keep lacking it, and a plain enable that wrote globally puts a
// service into projects nobody asked to change.
//
// Two projects in one installation is the smallest arrangement where either can
// be seen.
func TestGlobalServiceReachesTheNextProject(t *testing.T) {
	install := newInstallation(t)

	first := install.project("e2eglobalfirst")
	first.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2eglobalfirst.test",
	)
	first.run(20*time.Minute, "start")

	// Enabled for everyone, from inside one project.
	first.run(25*time.Minute, "service:enable", "--global", "phpmyadmin")
	requireContains(t, readFile(t, first.generated("docker-compose.yml")), "phpmyadmin:",
		"the project the command was run from")

	// A project created afterwards inherits it, because that is what --global
	// means and the only way to tell it apart from a local enable.
	second := install.project("e2eglobalsecond")
	second.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2eglobalsecond.test",
	)
	second.run(20*time.Minute, "start")

	requireContains(t, readFile(t, second.generated("docker-compose.yml")), "phpmyadmin:",
		"a project created after a global enable")
}

// TestLocalServiceStaysLocal is the other direction, and the more dangerous one.
//
// A plain `service:enable` that wrote globally would quietly add a service to
// every project made afterwards — on a machine with a dozen of them, discovered
// as containers nobody asked for.
func TestLocalServiceStaysLocal(t *testing.T) {
	install := newInstallation(t)

	first := install.project("e2elocalfirst")
	first.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2elocalfirst.test",
	)
	first.run(20*time.Minute, "start")

	first.run(25*time.Minute, "service:enable", "phpmyadmin")
	requireContains(t, readFile(t, first.generated("docker-compose.yml")), "phpmyadmin:",
		"the project the command was run in")

	second := install.project("e2elocalsecond")
	second.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2elocalsecond.test",
	)
	second.run(20*time.Minute, "start")

	if generated := readFile(t, second.generated("docker-compose.yml")); strings.Contains(generated, "phpmyadmin:") {
		t.Errorf("a local service:enable reached a project created afterwards:\n%s", generated)
	}
}
