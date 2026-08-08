//go:build e2e

package e2e

import (
	"strings"
	"testing"
	"time"
)

// TestEnablingAServiceAddsItAndDisablingRemovesIt covers the command people use
// to change what a project is made of.
//
// The interesting part is not the configuration write — that is one line — but
// whether the generated compose file and the running containers follow. A
// service enabled in the configuration and absent from the stack fails exactly
// like a setting that never reached the file, and looks identical from outside:
// the command says nothing is wrong.
//
// phpmyadmin, because it attaches to the database this project already has and
// needs nothing else.
//
// The assertions stop at the generated compose file on purpose. Whether the
// container then runs depends on the machine: the published phpmyadmin image is
// amd64 only, so on an arm64 host it is created, started and killed by the
// kernel with "exec format error". That is the image's business, not madock's,
// and asserting it would make this test pass or fail by architecture.
func TestEnablingAServiceAddsItAndDisablingRemovesIt(t *testing.T) {
	p := newProject(t, "e2eservice")

	p.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2eservice.test",
	)
	p.run(20*time.Minute, "start")

	compose := p.generated("docker-compose.yml")
	if strings.Contains(readFile(t, compose), "phpmyadmin:") {
		t.Fatal("this test assumes phpmyadmin is off to begin with")
	}

	// service:enable rebuilds the project itself, so this is the slow call.
	p.run(25*time.Minute, "service:enable", "phpmyadmin")

	requireContains(t, readFile(t, compose), "phpmyadmin:", "the compose file after enabling phpmyadmin")

	p.run(25*time.Minute, "service:disable", "phpmyadmin")

	after := readFile(t, compose)
	if strings.Contains(after, "phpmyadmin:") {
		t.Errorf("phpmyadmin is still in the compose file after being disabled:\n%s", after)
	}

	// And gone from the machine, not merely from the file. A container still
	// running for a service nobody asked for is the half of this that costs
	// memory rather than confusion.
	stopped := p.run(3*time.Minute, "status")
	if strings.Contains(stopped, "phpmyadmin") {
		t.Errorf("phpmyadmin is still reported after being disabled:\n%s", stopped)
	}
}

// TestEnablingAServiceThatDoesNotExistSaysSo pins the answer to a typo: say so,
// exit non-zero, and above all do not spend minutes rebuilding the project as
// if something had been enabled.
func TestEnablingAServiceThatDoesNotExistSaysSo(t *testing.T) {
	p := newProject(t, "e2ebadservice")

	p.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2ebadservice.test",
	)

	// Deliberately short: the point is that it returns without rebuilding
	// anything, and a rebuild would take far longer than this.
	out, err := p.tryRun(90*time.Second, "service:enable", "nosuchservice")
	if err == nil {
		t.Errorf("service:enable accepted a name that does not exist:\n%s", out)
	}
	requireContains(t, strings.ToLower(out), "doesn't exist", "the answer to an unknown service name")
}
