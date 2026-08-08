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
// Every debug command writes php/xdebug/*. On a project with no PHP container
// that set a value nothing reads, rebuilt the project, and reported success —
// so debugging was absent and the command that was meant to arrange it said
// otherwise. Wiring the other languages up is real work (their debuggers listen
// rather than connect out, so each needs a published port, an allocation and a
// process started under the debugger); saying so is one line.
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
	requireContains(t, strings.ToLower(out), "only wired up for php", "the answer on a project without PHP")

	if readFile(t, p.generated("docker-compose.yml")) != before {
		t.Error("debug:enable changed the generated configuration of a project it cannot debug")
	}

	// And the setting itself is untouched, so nothing reads back as debuggable.
	config := p.run(2*time.Minute, "config:list")
	if strings.Contains(config, "php/xdebug/enabled true") {
		t.Errorf("xdebug was switched on for a project with no PHP:\n%s", config)
	}
}
