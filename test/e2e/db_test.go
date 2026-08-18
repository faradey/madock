//go:build e2e

package e2e

import (
	"regexp"
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

	// Asserted line by line, not by searching the whole output, and that is not
	// fussiness: on this project the generated password is `db` — two characters,
	// the same as the database name, the user and the host — so "the output does
	// not contain the password" is false however well the masking works. The
	// first version of this test failed exactly there, on `host: db`.
	if line := valueLine(t, info, "password"); strings.Contains(line, password) || !strings.Contains(line, "set (") {
		t.Errorf("db:info did not describe the database password: %q", line)
	}
	if line := valueLine(t, info, "root password"); strings.Contains(line, rootPassword) || !strings.Contains(line, "set (") {
		t.Errorf("db:info did not describe the database root password: %q", line)
	}

	// And asked by name, it prints them — the flag is the whole point of the
	// default being the other way.
	shown := p.run(1*time.Minute, "db:info", "--show-secrets")
	if line := valueLine(t, shown, "password"); line != "password: "+password {
		t.Errorf("db:info --show-secrets did not print the password: %q", line)
	}
	if line := valueLine(t, shown, "root password"); line != "root password: "+rootPassword {
		t.Errorf("db:info --show-secrets did not print the root password: %q", line)
	}
}

// ansi matches the colour escapes madock writes around every printed line.
var ansi = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// valueLine returns the "<key>: <value>" line for one key, stripped of colour
// and surrounding space.
//
// Per-line, because the interesting assertions here are about one line each and
// a short secret is a substring of half the output. Fails the test when the key
// is absent, since every caller is asserting about a line it expects to be
// there.
func valueLine(t *testing.T, out, key string) string {
	t.Helper()

	for _, line := range strings.Split(ansi.ReplaceAllString(out, ""), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, key+":") {
			return line
		}
	}

	t.Fatalf("no %q line in:\n%s", key, out)
	return ""
}
