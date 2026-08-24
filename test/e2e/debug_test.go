//go:build e2e

package e2e

import (
	"strings"
	"testing"
	"time"
)

// TestDebugRefusesWhereItCannotWork covers the half of debugging that can be
// tested cheaply, and it is the half that was wrong.
//
// Every debug command used to write php/xdebug/*. On a project with no PHP
// container that set a value nothing reads, rebuilt the project, and reported
// success — so debugging was absent and the command that was meant to arrange
// it said otherwise. Node is wired up now; Python, Ruby and Go are not, and
// their debuggers listen rather than connect out, so each still needs a
// published port, an allocation and a process started under the debugger.
//
// This project has none of them, which is what makes it the cheap way to test
// that the command refuses instead of pretending.
//
// The PHP path is not tested here. It needs a project with a PHP container,
// which the cheap shape does not have, so it belongs with the platform tests
// that build a PHP image anyway.
func TestDebugRefusesWhereItCannotWork(t *testing.T) {
	p := newProject(t, "e2edebug")

	p.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2edebug.test",
	)
	p.run(20*time.Minute, "start")

	before := readFile(t, p.generated("docker-compose.yml"))

	// Deliberately short. A rebuild would take far longer, and not rebuilding is
	// half the point: the old behaviour spent minutes to change nothing.
	out, _ := p.tryRun(90*time.Second, "debug:enable")
	requireContains(t, strings.ToLower(out), "wired up for php and node", "the answer on a project with neither")
	requireContains(t, strings.ToLower(out), "nothing was changed", "the refusal saying it did nothing")

	if readFile(t, p.generated("docker-compose.yml")) != before {
		t.Error("debug:enable changed the generated configuration of a project it cannot debug")
	}

	// And the setting itself is untouched, so nothing reads back as debuggable.
	config := p.run(2*time.Minute, "config:list")
	if strings.Contains(config, "php/xdebug/enabled true") {
		t.Errorf("xdebug was switched on for a project with no PHP:\n%s", config)
	}
	if strings.Contains(config, "nodejs/debug/enabled true") {
		t.Errorf("the node debugger was switched on for a project with no node container:\n%s", config)
	}
}

// TestDebugRefusesNodeInsideAnotherContainer is the refusal that looks most like
// it should have worked.
//
// `nodejs/embedded/enabled` runs node in the application container instead of
// one of its own. Node is right there, so asking to debug it is reasonable — and
// it cannot work, because the debugger listens and there is no container of its
// own to publish a port from. Answering "this project runs neither" and stopping
// would be true and useless; it has to name the setting that gives node a
// container.
//
// Cheap on purpose: no rebuild, and the refusal is read straight off the
// command.
func TestDebugRefusesNodeInsideAnotherContainer(t *testing.T) {
	p := newProject(t, "e2edebugemb")

	p.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2edebugemb.test",
	)
	p.run(2*time.Minute, "config:set", "-n", "nodejs/embedded/enabled", "-v", "true")

	out, _ := p.tryRun(90*time.Second, "debug:enable")

	requireContains(t, strings.ToLower(out), "nodejs/embedded/enabled",
		"the refusal naming the setting that put node where it cannot be debugged")
	requireContains(t, strings.ToLower(out), "nodejs/enabled",
		"and the one that would give it a container of its own")

	config := p.run(2*time.Minute, "config:list")
	if strings.Contains(config, "nodejs/debug/enabled true") {
		t.Errorf("a switch was turned on that nothing renders:\n%s", config)
	}
}
