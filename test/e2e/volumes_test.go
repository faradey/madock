//go:build e2e

package e2e

import (
	"strings"
	"testing"
	"time"
)

// TestSnapshotsAndClonesCarryEveryNamedVolume — a snapshot used to hold the
// databases and the files and nothing else. A search index, a persisted Redis
// or a broker's queues stayed behind, so a restore paired today's index with
// last week's catalogue and a clone came up empty everywhere but in MySQL.
//
// Redis stands in for every such volume: it is the cheapest service with state
// of its own, and redis/persist is what gives it a named volume at all. Without
// that key its /data sits in the image's anonymous volume, which no snapshot
// sees.
func TestSnapshotsAndClonesCarryEveryNamedVolume(t *testing.T) {
	install := newInstallation(t)
	source := install.project("e2evols")

	source.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2evols.test",
	)
	source.run(2*time.Minute, "config:set", "-n", "redis/enabled", "-v", "true")
	source.run(2*time.Minute, "config:set", "-n", "redis/persist", "-v", "true")
	source.run(25*time.Minute, "start")

	redis := func(p *project, args ...string) string {
		return strings.TrimSpace(p.run(2*time.Minute, append([]string{"cli", "--service", "redisdb", "redis-cli"}, args...)...))
	}

	redis(source, "SET", "probe", "before-snapshot")
	source.run(10*time.Minute, "snapshot:create", "-n", "volumes")
	redis(source, "SET", "probe", "after-snapshot")

	source.run(20*time.Minute, "snapshot:restore", "--name", "volumes")
	if got := redis(source, "GET", "probe"); got != "before-snapshot" {
		t.Errorf("after the restore Redis holds %q, want the value from before the snapshot", got)
	}

	source.run(30*time.Minute, "project:clone", "-n", "e2evolscopy", "-s=-copy")
	clone := install.project("e2evolscopy")
	clone.run(20*time.Minute, "start")

	if got := redis(clone, "GET", "probe"); got != "before-snapshot" {
		t.Errorf("the clone's Redis holds %q, want the source's value", got)
	}

	// Copied, not shared: a write in the clone must stay there.
	redis(clone, "SET", "probe", "written-in-clone")
	if got := redis(source, "GET", "probe"); got != "before-snapshot" {
		t.Errorf("a write in the clone's Redis reached the source: %q", got)
	}
}
