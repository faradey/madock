//go:build e2e

package e2e

import (
	"testing"
	"time"
)

// TestDatabaseIsReachable proves the database commands find their container and
// authenticate against it.
//
// It is deliberately not a test of SQL. Everything that usually breaks here is
// upstream of the query: which container the command picks, which host it
// connects to, which credentials it uses. `db:execute` failing with "No such
// container" is the shape of every database defect this project has had.
func TestDatabaseIsReachable(t *testing.T) {
	p := newProject(t, "e2edb")

	p.run(3*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2edb.test",
	)
	p.run(20*time.Minute, "start")

	out := p.query("SELECT 1 AS madock_probe")

	requireContains(t, out, "madock_probe", "db:execute should print the column")
	requireContains(t, out, "1", "db:execute should print the value")

	// A round trip, because a SELECT of a constant would also succeed against
	// the wrong database. Writing and then finding it again is what proves the
	// two commands landed in the same place.
	p.freshTable("madock_e2e_probe", "(id INT)")
	tables := p.query("SHOW TABLES")
	requireContains(t, tables, "madock_e2e_probe", "the table created a moment ago")

	// db:info is what people read before filing a bug, so it has to agree with
	// what the other commands actually used. The database is named `db` by
	// default — after the project would be the obvious guess, and it is wrong.
	info := p.run(1*time.Minute, "db:info")
	requireContains(t, info, "type: MYSQL", "db:info should report the engine")
	requireContains(t, info, "host: db", "db:info should report the host the commands connect to")
}
