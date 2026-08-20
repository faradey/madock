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

	// The passwords are printed, because this is the edition that manages a
	// developer's own laptop: `db:info` is run here to copy a password into a
	// database client, and the config file it comes from is two directories
	// away, so withholding it would add a flag to every such use and protect
	// nothing.
	//
	// The paid edition answers the other way and its own suite pins that. There
	// the same command prints a shared database's **root** password — which
	// reaches every other project's schema on that server — and the output goes
	// into a ticket, a screen share or a CI log. Same command, different machine;
	// the edition decides, through fmtc.HideSecretsByDefault.
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

	// Compared as whole values, not searched for anywhere in the output. This
	// project's credentials are `db` and `password` — two and eight characters,
	// the same as the database name and a substring of the word "password" in
	// madock's own label — so "the output contains the secret" is true whatever
	// the command actually printed. Two versions of this test failed on exactly
	// that, on `host: db` and then on `root password: set (8)`. Reading the value
	// off its own line is what makes the assertion mean anything, in either
	// direction.
	if got := valueOf(t, info, "password"); got != password {
		t.Errorf("db:info printed %q as the database password, want %q", got, password)
	}
	if got := valueOf(t, info, "root password"); got != rootPassword {
		t.Errorf("db:info printed %q as the database root password, want %q", got, rootPassword)
	}

	// --show-secrets is accepted and changes nothing here. It has to keep
	// working: the same command line is written for both editions, and one that
	// errored on the flag would make every such script edition-specific.
	shown := p.run(1*time.Minute, "db:info", "--show-secrets")
	if got := valueOf(t, shown, "password"); got != password {
		t.Errorf("db:info --show-secrets printed %q as the password, want %q", got, password)
	}
	if got := valueOf(t, shown, "root password"); got != rootPassword {
		t.Errorf("db:info --show-secrets printed %q as the root password, want %q", got, rootPassword)
	}
}

// ansi matches the colour escapes madock writes around every printed line.
var ansi = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// valueOf returns what one "<key>: <value>" line says, stripped of colour and
// surrounding space.
//
// The value alone, because every assertion here is an equality against a secret,
// and a secret that happens to be a substring of the label or of another field is
// exactly what a containment check cannot survive. Fails the test when the key is
// absent, since every caller expects the line to be there.
func valueOf(t *testing.T, out, key string) string {
	t.Helper()

	for _, line := range strings.Split(ansi.ReplaceAllString(out, ""), "\n") {
		line = strings.TrimSpace(line)
		if value, found := strings.CutPrefix(line, key+":"); found {
			return strings.TrimSpace(value)
		}
	}

	t.Fatalf("no %q line in:\n%s", key, out)
	return ""
}
