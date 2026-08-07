//go:build e2e

package e2e

import (
	"os"
	"strings"
	"testing"
	"time"
)

// TestConfigChangeReachesGeneratedFiles covers the gap that produced a real
// support question: a value set with `config:set` was only picked up by a full
// `rebuild`, so a `restart` after changing the config appeared to do nothing
// and the setting looked like it had been ignored.
//
// restart_policy is the cheapest knob that proves it. It renders straight into
// the compose file and changing it pulls no images, so the test says something
// about the mechanism rather than about the network.
func TestConfigChangeReachesGeneratedFiles(t *testing.T) {
	p := newProject(t, "e2econf")

	p.run(3*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2econf.test",
	)

	compose := p.generated("docker-compose.yml")
	requireContains(t, readFile(t, compose), "restart: no", "the default restart policy")

	p.run(2*time.Minute, "config:set", "-n", "restart_policy", "-v", "always")

	// config:set writes the project config. Nothing is regenerated until a
	// command that touches the runtime runs — which is the point: the value has
	// to survive the trip.
	listed := p.run(2*time.Minute, "config:list")
	requireContains(t, listed, "always", "config:list after config:set")

	p.run(20*time.Minute, "start")

	generated := readFile(t, compose)
	requireContains(t, generated, "restart: always", "the compose file after start")
	if strings.Contains(generated, "restart: no") {
		t.Errorf("the old restart policy is still in the generated compose file:\n%s", generated)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(content)
}
