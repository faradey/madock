//go:build e2e

package e2e

import (
	"strings"
	"testing"
	"time"
)

// TestLogsSelectTheServiceAskedFor covers the flag that decides which container
// you are reading.
//
// `logs -s` picking the wrong container is worse than failing: the output looks
// like logs, so the reader concludes the service is quiet and goes looking
// somewhere else. It is the command people reach for when something is already
// wrong, which is the worst moment to be shown the wrong thing.
//
// The database is the anchor, because its startup output is unmistakable and
// every project in this suite has one.
func TestLogsSelectTheServiceAskedFor(t *testing.T) {
	p := newProject(t, "e2elogs")

	p.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2elogs.test",
	)
	p.run(20*time.Minute, "start")

	// The database says a good deal on startup; wait until it has.
	var dbLogs string
	deadline := time.Now().Add(2 * time.Minute)
	for {
		dbLogs = p.run(2*time.Minute, "logs", "-s", "db")
		if strings.Contains(dbLogs, "mariadb") || strings.Contains(dbLogs, "InnoDB") || strings.Contains(dbLogs, "mysqld") {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("the database never wrote anything recognisable to its log:\n%s", dbLogs)
		}
		time.Sleep(5 * time.Second)
	}

	// Another service must not be showing the database's output. An empty log
	// satisfies this and is a fine answer — the wrong log is not.
	nginxLogs := p.run(2*time.Minute, "logs", "-s", "nginx")
	if strings.Contains(nginxLogs, "InnoDB") || strings.Contains(nginxLogs, "mariadbd") {
		t.Errorf("logs -s nginx showed the database's output:\n%s", nginxLogs)
	}

	// A service that does not exist has to say so. Falling back to the main one
	// would be the quiet failure this test is about, one step further along.
	out, err := p.tryRun(2*time.Minute, "logs", "-s", "nosuchservice")
	if err == nil {
		t.Errorf("logs accepted a service that does not exist:\n%s", out)
	}
}
