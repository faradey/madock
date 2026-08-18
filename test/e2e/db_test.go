//go:build e2e

package e2e

import (
	"strings"
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

	// The passwords are described, not printed. This command is run to find a
	// host and a port, and its output then lives in scrollback, in a CI log, in
	// a screenshot and in the issue somebody pastes it into — a radius the
	// config file it reads from does not have. On a project borrowing a shared
	// database the root password printed here is the **provider's**, which
	// reaches every other project's schema on that server.
	//
	// The values are read back from the configuration rather than set to
	// something recognisable first, and that is not a stylistic choice: the
	// database initialises its data directory from the credentials in the
	// generated compose file on its first start, so writing new ones afterwards
	// gives a project whose config and whose container disagree — measured, as
	// 60 rounds of "Access denied for user 'root'" and a test that hung for five
	// minutes before failing. What is under test here is the printing, and the
	// generated passwords are the honest input for it.
	password := p.configValue("db/password")
	rootPassword := p.configValue("db/root_password")
	if password == "" || rootPassword == "" {
		t.Fatalf("the project has no generated database passwords to check against: %q / %q", password, rootPassword)
	}

	if strings.Contains(info, password) {
		t.Errorf("db:info printed the database password in full:\n%s", info)
	}
	if strings.Contains(info, rootPassword) {
		t.Errorf("db:info printed the database root password in full:\n%s", info)
	}
	requireContains(t, info, "password: set (", "db:info should still say a password is set, and how long it is")

	// And asked by name, it prints them — the flag is the whole point of the
	// default being the other way.
	shown := p.run(1*time.Minute, "db:info", "--show-secrets")
	requireContains(t, shown, password, "db:info --show-secrets should print the password")
	requireContains(t, shown, rootPassword, "db:info --show-secrets should print the root password")
}
