//go:build e2e

package e2e

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// TestSecondDatabaseImportsIntoItself is the `--service db2` twin the import
// side never had.
//
// Export against db2 is covered; import is not, and the two are not symmetrical
// in what going wrong costs. An export that reads the wrong server produces a
// file nobody trusts. An import that writes to the wrong server **replaces a
// database that was not being restored** — the first one, the project's real
// data, overwritten by the contents of the second while the command reports a
// successful restore.
//
// It also runs the two import flags nothing else does: `--reset-gtid`, which
// issues RESET MASTER before loading, and `-u`, which is the container user the
// client runs as rather than the database login.
func TestSecondDatabaseImportsIntoItself(t *testing.T) {
	p := newProject(t, "e2edb2imp")

	p.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2edb2imp.test",
	)

	// Its own credentials, for the reason the sibling test gives: identical ones
	// would let a command that connects to the wrong server succeed.
	p.run(2*time.Minute, "config:set", "-n", "db2/enabled", "-v", "true")
	p.run(2*time.Minute, "config:set", "-n", "db2/database", "-v", "second")
	p.run(2*time.Minute, "config:set", "-n", "db2/user", "-v", "second_user")
	p.run(2*time.Minute, "config:set", "-n", "db2/password", "-v", "second_pw")
	p.run(2*time.Minute, "config:set", "-n", "db2/root_password", "-v", "second_root")

	p.run(25*time.Minute, "start")

	waitForSecondDatabase(t, p)

	// Two servers, two tables, no overlap in the names — so any row appearing
	// where it does not belong names the mistake by itself.
	p.querySecond("DROP TABLE IF EXISTS second_probe")
	p.querySecond("CREATE TABLE second_probe (note VARCHAR(32))")
	p.querySecond("INSERT INTO second_probe VALUES ('before-dump')")

	p.freshTable("first_probe", "(note VARCHAR(32))")
	p.query("INSERT INTO first_probe VALUES ('untouched')")

	dump := dumpFile(t, p.run(10*time.Minute, "db:export", "--service", "db2", "-n", "twin", "--json"))

	// Written after the dump, so the restore has something to undo. Without it
	// the test would pass against an import that did nothing at all.
	p.querySecond("INSERT INTO second_probe VALUES ('after-dump')")

	p.run(10*time.Minute, "db:import", "--force", "--service", "db2", dump)

	restored := p.querySecond("SELECT note FROM second_probe")
	requireContains(t, restored, "before-dump", "the row the db2 dump contained")
	if strings.Contains(restored, "after-dump") {
		t.Errorf("the row written to db2 after the dump survived the import:\n%s", restored)
	}

	// The first database is the one with everything to lose. It must still hold
	// its own table and must not have acquired db2's.
	firstRows := p.query("SELECT note FROM first_probe")
	requireContains(t, firstRows, "untouched", "the first database after an import aimed at db2")

	firstTables := p.query("SHOW TABLES")
	if strings.Contains(firstTables, "second_probe") {
		t.Errorf("importing into db2 wrote db2's table into db:\n%s", firstTables)
	}

	// --reset-gtid issues RESET MASTER before loading, and madock's database
	// images run with binary logging off — the server answers "Binlog closed,
	// cannot RESET MASTER" and used to take the whole import down with it, with
	// the server's message swallowed and only "exit status 1" printed. The flag
	// therefore could not work on any default project, which is what nothing
	// ever ran it long enough to notice.
	//
	// The premise is asserted rather than assumed: if a later madock turns
	// binary logging on, this test should say so instead of quietly checking a
	// branch that no longer runs.
	requireContains(t, p.querySecond("SELECT @@log_bin"), "0",
		"binary logging off, which is what makes the reset impossible here")

	p.querySecond("INSERT INTO second_probe VALUES ('after-first-import')")
	resetOut := p.run(10*time.Minute, "db:import", "--force", "--reset-gtid", "--service", "db2", dump)

	// It has to say the reset did not happen. Carrying on silently would leave
	// the caller believing the GTID state was cleared.
	requireContains(t, resetOut, "Binary logging is off",
		"an import that could not reset the GTID state saying so")

	afterReset := p.querySecond("SELECT note FROM second_probe")
	requireContains(t, afterReset, "before-dump", "the row restored by an import with --reset-gtid")
	if strings.Contains(afterReset, "after-first-import") {
		t.Errorf("--reset-gtid import did not replace the rows written since the dump:\n%s", afterReset)
	}

	// -u is the container user the mysql client runs as, not the database
	// login. The default is `mysql`; root is the one people reach for when a
	// permission gets in the way, and nothing has ever run it.
	p.run(10*time.Minute, "db:import", "--force", "--service", "db2", "-u", "root", dump)
	requireContains(t, p.querySecond("SELECT note FROM second_probe"), "before-dump",
		"the row restored by an import running as root in the container")
}

// waitForSecondDatabase blocks until db2 answers, the way the first database is
// waited for. A container that exists is not a server accepting connections, and
// snapshot:create and snapshot:restore both stop it.
func waitForSecondDatabase(t *testing.T, p *project) {
	t.Helper()
	p.queryOn("db2", "SELECT 1")
}

// querySecond is query() against db2.
func (p *project) querySecond(sql string) string {
	p.t.Helper()
	return p.queryOn("db2", sql)
}

// dumpFile returns the path db:export --json reported.
func dumpFile(t *testing.T, exportOutput string) string {
	t.Helper()

	var payload struct {
		Success bool `json:"success"`
		Data    struct {
			File string `json:"file"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(jsonPart(exportOutput)), &payload); err != nil {
		t.Fatalf("db:export --json did not decode: %v\n%s", err, exportOutput)
	}
	if !payload.Success {
		t.Fatalf("db:export --json reported failure:\n%s", exportOutput)
	}
	if payload.Data.File == "" {
		t.Fatalf("db:export --json reported no file:\n%s", exportOutput)
	}
	return payload.Data.File
}
