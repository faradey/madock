//go:build e2e

package e2e

import (
	"strings"
	"testing"
	"time"
)

// The database is the part of a project that reserves memory before it holds
// any data, and until db/memory existed the numbers were written into my.cnf by
// hand — 768 MB for every MySQL, unreachable by any setting. These tests run
// the budget through to a live server and ask the server what it got, because
// the one thing a golden file cannot show is whether the container starts at
// all with the settings we generated.
//
// Each engine gets its own project. Switching db/type on an existing one leaves
// the first engine's data directory in the volume and the second refuses to
// start on it — which is exactly why --db-type exists, and why it is used here
// at setup time rather than with config:set afterwards.

// TestDatabaseMemoryReachesMysql checks the default and then a raised budget.
//
// The default has to come out at what used to be hard-coded, or every existing
// project's database is resized by an upgrade nobody asked for.
func TestDatabaseMemoryReachesMysql(t *testing.T) {
	p := newProject(t, "e2edbmemmysql")

	p.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2edbmemmysql.test",
	)
	p.run(20*time.Minute, "start")

	// 768M split two thirds / one third: the two numbers that were in my.cnf.
	requireContains(t, p.query("SHOW VARIABLES LIKE 'innodb_buffer_pool_size'"),
		"536870912", "the default budget should leave the buffer pool where it was")

	p.run(2*time.Minute, "config:set", "-n", "db/memory", "-v", "1536M")
	p.run(20*time.Minute, "restart")

	requireContains(t, p.query("SHOW VARIABLES LIKE 'innodb_buffer_pool_size'"),
		"1073741824", "raising db/memory should raise the buffer pool")
}

// TestDatabaseMemoryReachesPostgresql is the one that proves the command line
// generated for postgres is a command line postgres accepts.
//
// PostgreSQL had never been given a memory setting at all and sat on its stock
// 128MB. The setting arrives as `-c shared_buffers=...` on the server's own
// command, which replaces the image's default command — so a mistake here is
// not a wrong number, it is a container that does not come up.
func TestDatabaseMemoryReachesPostgresql(t *testing.T) {
	p := newProject(t, "e2edbmempg")

	p.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--db-type=postgresql",
		// Pinned, and not to the newest. This test is about memory, and
		// postgres 18 moved its data directory: the image expects the volume on
		// /var/lib/postgresql while madock still mounts /var/lib/postgresql/data,
		// so an 18 container refuses to start whatever memory it is given. That
		// is its own defect, and it should fail its own test rather than this one.
		"--db=16",
		"--hosts=e2edbmempg.test",
	)
	p.run(20*time.Minute, "start")

	requireContains(t, p.run(3*time.Minute, "status"), "db running",
		"postgres must actually start with the generated command")

	// A quarter of 768M. postgres reports it in 8kB pages by default, so ask
	// for it in bytes and compare against the number the budget produces.
	shared := p.query("SELECT setting, unit FROM pg_settings WHERE name = 'shared_buffers'")
	if !strings.Contains(shared, "24576") {
		t.Errorf("shared_buffers is not the quarter of the budget that was generated:\n%s", shared)
	}

	requireContains(t, p.query("SELECT setting FROM pg_settings WHERE name = 'effective_cache_size'"),
		"73728", "effective_cache_size should be three quarters of the budget")
}

// TestDatabaseMemoryReachesMongodb covers the engine that is dangerous when
// left alone: WiredTiger sizes its cache from the RAM it can see, which is the
// host's rather than the container's limit, so one mongod on a large machine
// reserves gigabytes nobody allocated to it.
func TestDatabaseMemoryReachesMongodb(t *testing.T) {
	p := newProject(t, "e2edbmemmongo")

	p.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--db-type=mongodb",
		"--hosts=e2edbmemmongo.test",
	)
	p.run(20*time.Minute, "start")

	requireContains(t, p.run(3*time.Minute, "status"), "db running",
		"mongod must actually start with --wiredTigerCacheSizeGB")

	// Half of 768M, which mongod reports back in bytes.
	stats := p.query(`db.serverStatus().wiredTiger.cache["maximum bytes configured"]`)
	requireContains(t, stats, "402653184", "the WiredTiger cache should be half the budget")
}
