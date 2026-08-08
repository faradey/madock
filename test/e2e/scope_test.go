//go:build e2e

package e2e

import (
	"strings"
	"testing"
	"time"
)

// TestScopesKeepTheirOwnValues covers the feature that lets one project hold
// several configurations — a branch with a different PHP version, a staging
// shape, a customer's variant — and switch between them.
//
// The risk it carries is quiet: a value set while one scope is active and read
// while another is, or a scope that switches without the generated files
// following. Either produces a project that is not what its configuration says,
// and nothing reports it, because every individual command succeeded.
//
// restart_policy again: it renders straight into the compose file, so what the
// scope did is visible in a generated file rather than only in the config.
func TestScopesKeepTheirOwnValues(t *testing.T) {
	p := newProject(t, "e2escope")

	p.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2escope.test",
	)
	p.run(20*time.Minute, "start")

	compose := p.generated("docker-compose.yml")
	requireContains(t, readFile(t, compose), "restart: no", "the default scope's value")

	// One scope runs at a time: a scope is a different shape of the same
	// project, not a second copy of it. The containers of the scope being left
	// have to go down before the next one comes up, or the two claim the same
	// published ports and the second start fails with "port is already
	// allocated".
	p.run(5*time.Minute, "stop")

	// Adding a scope activates it, which is why the value set next belongs to
	// the new one and not to default.
	p.run(2*time.Minute, "scope:add", "staging")
	requireContains(t, p.run(2*time.Minute, "scope:list"), "staging", "the new scope in the list")

	p.run(2*time.Minute, "config:set", "-n", "restart_policy", "-v", "always")
	p.run(20*time.Minute, "start")
	requireContains(t, readFile(t, compose), "restart: always", "the compose file under the new scope")

	// Back to default. The value set in staging must not have followed.
	p.run(5*time.Minute, "stop")
	p.run(2*time.Minute, "scope:set", "default")
	p.run(20*time.Minute, "start")

	back := readFile(t, compose)
	requireContains(t, back, "restart: no", "the default scope after switching back")
	if strings.Contains(back, "restart: always") {
		t.Errorf("a value set in another scope leaked into default:\n%s", back)
	}

	// And staging still holds its own, rather than having been overwritten by
	// the trip through default.
	p.run(5*time.Minute, "stop")
	p.run(2*time.Minute, "scope:set", "staging")
	p.run(20*time.Minute, "start")
	requireContains(t, readFile(t, compose), "restart: always", "the scope's value after switching away and back")
}
