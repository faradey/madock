//go:build e2e

package e2e

import (
	"os"
	"strings"
	"testing"
	"time"
)

// TestSecondDatabaseIsItsOwnDatabase walks the parallel path that every db:*
// command has and that nothing has ever run.
//
// db2 is a second server with its own container, its own credentials and its
// own schema. A golden case pins that its credentials are not read from db/* —
// the mistake that would otherwise be invisible, since the two would happily
// connect using each other's password if the values matched. Nothing checks the
// live half: that `--service db2` actually reaches the other server, and that
// writing to one does not appear in the other.
//
// Getting this wrong in either direction is expensive. A command that silently
// falls back to db writes a project's second database into its first; one that
// reads db while claiming db2 reports the wrong data as the right data.
func TestSecondDatabaseIsItsOwnDatabase(t *testing.T) {
	// Quarantined 2026-08-27, and not for being flaky — for failing on something
	// that is not its own subject.
	//
	// Measured: two failures in four runs of the same tree, always here, always
	// `ERROR 1130 (HY000): Host '…' is not allowed to connect to this MariaDB
	// server` from **db**, the first database, which this test does not
	// configure. The container log says why, and says it in one line:
	//
	//	[Entrypoint]: MariaDB upgrade (mariadb-upgrade or creating healthcheck
	//	users) required, but skipped due to $MARIADB_AUTO_UPGRADE setting
	//
	// There is no "Initializing database files" above it and no "Creating
	// user". The data directory was already there, made by another version, so
	// the entrypoint skipped initialisation altogether and never created the
	// account the project connects with. That is permanent, not a race: the
	// harness's `refusalGrace` exists for a server that answers before it has
	// finished deciding, and it spends a minute here waiting for something that
	// is never going to happen.
	//
	// Where a fresh project gets a populated data directory is the open
	// question. Project names were the first suspect and are ruled out — 51
	// tests, 51 distinct names. The volume is `dbdata` in
	// docker/snippets/docker-compose/db.yml, so the next measurement is which
	// compose project it is created under on a runner, and that needs a
	// disposable daemon rather than a laptop.
	//
	// Run it deliberately with MADOCK_E2E_QUARANTINED=yes.
	if os.Getenv("MADOCK_E2E_QUARANTINED") != "yes" {
		t.Skip("quarantined: fails on a pre-existing MariaDB data directory, not on its own subject — MADOCK-BACKLOG.md, and MADOCK_E2E_QUARANTINED=yes to run it")
	}

	p := newProject(t, "e2edb2")

	p.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2edb2.test",
	)

	// Its own credentials on purpose. Identical ones would let a command that
	// connects to the wrong server succeed, which is the defect being ruled out.
	p.run(2*time.Minute, "config:set", "-n", "db2/enabled", "-v", "true")
	p.run(2*time.Minute, "config:set", "-n", "db2/database", "-v", "second")
	p.run(2*time.Minute, "config:set", "-n", "db2/user", "-v", "second_user")
	p.run(2*time.Minute, "config:set", "-n", "db2/password", "-v", "second_pw")
	p.run(2*time.Minute, "config:set", "-n", "db2/root_password", "-v", "second_root")

	p.run(25*time.Minute, "start")

	requireContains(t, p.run(3*time.Minute, "status"), "db2 running", "the second database container")

	// Wait for the second server the same way the first one is waited for.
	var err error
	var out string
	deadline := time.Now().Add(3 * time.Minute)
	for {
		out, err = p.tryRun(time.Minute, "db:execute", "--service", "db2", "SELECT 1")
		if err == nil || time.Now().After(deadline) {
			break
		}
		time.Sleep(5 * time.Second)
	}
	if err != nil {
		t.Fatalf("db:execute --service db2 never succeeded: %v\n%s", err, out)
	}

	p.run(time.Minute, "db:execute", "--service", "db2", "CREATE TABLE only_in_second (note VARCHAR(32))")
	p.run(time.Minute, "db:execute", "--service", "db2", "INSERT INTO only_in_second VALUES ('second')")

	inSecond := p.run(time.Minute, "db:execute", "--service", "db2", "SELECT note FROM only_in_second")
	requireContains(t, inSecond, "second", "the row written to the second database")

	// The first database must not have it — neither the row nor the table.
	firstTables := p.query("SHOW TABLES")
	if strings.Contains(firstTables, "only_in_second") {
		t.Errorf("a table created in db2 appeared in db:\n%s", firstTables)
	}

	// And the same in reverse, because a command reading the wrong server is as
	// bad as one writing to it.
	p.freshTable("only_in_first", "(note VARCHAR(32))")
	secondTables := p.run(time.Minute, "db:execute", "--service", "db2", "SHOW TABLES")
	if strings.Contains(secondTables, "only_in_first") {
		t.Errorf("a table created in db appeared in db2:\n%s", secondTables)
	}

	// An export of db2 is an export of db2. The file name carries the source, so
	// a dump of the wrong server is not even distinguishable afterwards.
	dump := dumpContents(t, p, p.run(10*time.Minute, "db:export", "--service", "db2", "-n", "second", "--json"))
	requireContains(t, dump, "only_in_second", "the second database's table in its own dump")
	if strings.Contains(dump, "only_in_first") {
		t.Error("the dump of db2 contains a table that only exists in db")
	}
}
