//go:build e2e

package e2e

import (
	"strings"
	"testing"
	"time"
)

// TestRecreatingADatabaseWaitsForItsInitialisation is the "MariaDB directory
// already full on a fresh project" case, reproduced on purpose.
//
// `setup` starts the containers, and the database image begins initialising
// its data directory: a temporary server, the account SQL, a stop, a real
// start. `rebuild` a second later took that container down in the middle of
// it. The data directory then had the `mysql` schema — so the next start
// skipped initialisation — and whichever accounts the script had reached:
// root@localhost with a password, no root@'%'. Every `db:execute` after that
// answered ERROR 1130, for as long as the volume lived.
//
// It was struck off the backlog as unreproducible because a developer machine
// finishes initialising before anyone can type the next command. CI, and a
// person on a new server following the README, are slower than that. Measured
// in CI on 2026-09-11 in two tests that do setup, config:set and start within
// the same ten seconds.
func TestRecreatingADatabaseWaitsForItsInitialisation(t *testing.T) {
	p := newProject(t, "e2edbinit")
	p.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2edbinit.test",
	)

	// No pause here, deliberately: the rebuild has to land while the
	// database is still initialising, which is the whole reproduction.
	p.run(20*time.Minute, "rebuild")

	// The account the image creates last is root@'%', and it is the one every
	// madock command connects as. If the initialisation was interrupted this
	// answers 1130 on every attempt, and the harness gives that a minute.
	if out := p.query("SELECT 1"); !strings.Contains(out, "1") {
		t.Errorf("the database does not answer after a rebuild that followed setup at once:\n%s", out)
	}

	// The log of the container that survived must show a completed
	// initialisation or none at all — never a start that found a directory
	// the previous container had half-written.
	log := p.run(2*time.Minute, "logs", "-s", "db")
	if strings.Contains(log, "Starting crash recovery") {
		t.Errorf("the database recovered a directory the previous container left half-written — the rebuild did not wait:\n%s", tail(log, 30))
	}
}

func tail(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}

	return strings.Join(lines, "\n")
}
