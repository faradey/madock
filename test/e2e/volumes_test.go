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

	source.run(setupTimeout, "setup", "-y",
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

// TestTurningPersistOnCarriesTheOldData — redis/persist moves Redis to a named
// volume, and until this the next start began it empty: the data stayed in the
// old container's anonymous volume. On a server that is a Packeton's download
// statistics or an application's queued jobs.
func TestTurningPersistOnCarriesTheOldData(t *testing.T) {
	p := newProject(t, "e2epersist")
	p.run(setupTimeout, "setup", "-y", "--platform=custom", "--language=none", "--hosts=e2epersist.test")
	p.run(2*time.Minute, "config:set", "-n", "redis/enabled", "-v", "true")
	p.run(25*time.Minute, "start")

	redis := func(args ...string) string {
		return strings.TrimSpace(p.run(2*time.Minute, append([]string{"cli", "--service", "redisdb", "redis-cli"}, args...)...))
	}
	redis("SET", "probe", "kept-across-persist")

	p.run(2*time.Minute, "config:set", "-n", "redis/persist", "-v", "true")
	out := p.run(25*time.Minute, "start")
	t.Logf("start after persist:\n%s", out)
	requireContains(t, out, "data carried", "the start that switched Redis to its named volume")
	if got := redis("GET", "probe"); got != "kept-across-persist" {
		t.Errorf("after redis/persist Redis holds %q, want the value written before", got)
	}

	// Once carried, a later start does it no more.
	if out := p.run(10*time.Minute, "start"); strings.Contains(out, "Carrying") {
		t.Errorf("a second start carried the data again:\n%s", out)
	}
}

// TestTurningPersistOnCarriesValkeyFromItsLayer — the image valkey/valkey
// declares no VOLUME, so without valkey/persist the data sits in the
// container's own layer and a recreate removes it with the container. That is
// how a production store lost 10 724 sessions on 2026-10-20: the first version
// of the carry looked only for an anonymous volume, found none, and let the
// start recreate the container.
func TestTurningPersistOnCarriesValkeyFromItsLayer(t *testing.T) {
	p := newProject(t, "e2evalkeylayer")
	p.run(setupTimeout, "setup", "-y", "--platform=custom", "--language=none", "--hosts=e2evalkeylayer.test")
	p.run(2*time.Minute, "config:set", "-n", "valkey/enabled", "-v", "true")
	// 9, the version the store ran: valkey/valkey:9 declares no VOLUME, the
	// 8.1 line madock defaults to does — and with it the data is in an
	// anonymous volume, which is the other case.
	p.run(2*time.Minute, "config:set", "-n", "valkey/version", "-v", "9")
	p.run(25*time.Minute, "start")

	valkey := func(args ...string) string {
		return strings.TrimSpace(p.run(2*time.Minute, append([]string{"cli", "--service", "valkeydb", "valkey-cli"}, args...)...))
	}
	valkey("SET", "probe", "kept-from-the-layer")

	p.run(2*time.Minute, "config:set", "-n", "valkey/persist", "-v", "true")
	out := p.run(25*time.Minute, "start")
	t.Logf("start after persist:\n%s", out)
	requireContains(t, out, "the container's own layer", "the start that moved Valkey to its named volume")
	if got := valkey("GET", "probe"); got != "kept-from-the-layer" {
		t.Errorf("after valkey/persist Valkey holds %q, want the value written before", got)
	}
}
