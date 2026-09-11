package docker

import (
	"strings"
	"testing"
	"time"
)

// TestInitInProgressReadsTheEntrypointsOwnMarkers pins the three log shapes
// the wait decides on.
//
// The middle one is the case this exists for: a MariaDB entrypoint that has
// started its temporary server and not yet said it is done. Recreating that
// container leaves a data directory with the `mysql` schema — so the next start
// skips initialisation — and whichever accounts the script had reached.
// Measured in CI on 2026-09-11: root@localhost with a password, no root@'%',
// and `db:execute` answering ERROR 1130 for as long as the volume lived.
func TestInitInProgressReadsTheEntrypointsOwnMarkers(t *testing.T) {
	cases := []struct {
		name string
		log  string
		want bool
	}{
		{"found an initialised directory, started straight away",
			"[Entrypoint]: MariaDB upgrade not required\n[Note] mariadbd: ready for connections.\n", false},
		{"mid-initialisation",
			"[Entrypoint]: Initializing database files\n[Entrypoint]: Temporary server started.\n", true},
		{"initialisation finished",
			"[Entrypoint]: Temporary server started.\n[Entrypoint]: Temporary server stopped\n[Entrypoint]: MariaDB init process done. Ready for start up.\n", false},
		{"postgres mid-initialisation",
			"initializing database system\nselecting default max_connections ... 100\n", true},
		{"postgres finished",
			"initializing database system\nPostgreSQL init process complete; ready for start up.\n", false},
		{"no log at all", "", false},
	}

	for _, c := range cases {
		if got := initInProgress(c.log); got != c.want {
			t.Errorf("%s: initInProgress = %v, want %v", c.name, got, c.want)
		}
	}
}

// TestTheWaitEndsWhenTheEntrypointSaysDone drives the wait with a log that
// finishes on the third read, and asserts it neither returns early nor waits
// for the timeout.
func TestTheWaitEndsWhenTheEntrypointSaysDone(t *testing.T) {
	reads := 0
	orig := containerLog
	containerLog = func(container string) string {
		reads++
		if !strings.Contains(container, "-db-1") {
			t.Errorf("asked about %s, which is not the database", container)
		}
		if reads < 3 {
			return "[Entrypoint]: Temporary server started.\n"
		}

		return "[Entrypoint]: Temporary server started.\n[Entrypoint]: MariaDB init process done.\n"
	}
	t.Cleanup(func() { containerLog = orig })

	started := time.Now()
	WaitForDatabaseInit("waitproject")

	if reads < 3 {
		t.Errorf("the wait gave up after %d reads while the entrypoint was still initialising", reads)
	}
	if time.Since(started) > 30*time.Second {
		t.Errorf("the wait ran to its timeout although the entrypoint said done on the third read")
	}
}
