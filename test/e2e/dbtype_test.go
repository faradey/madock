//go:build e2e

package e2e

import (
	"strings"
	"testing"
	"time"
)

// TestMariaDbTypeStartsADatabase is the field case: a project whose config says
// `MariaDB` has to come up with a database.
//
// Every compose snippet gates on one of three engine families — db.yml on
// "mysql", db-postgresql.yml on "postgresql", db-mongodb.yml on "mongodb" — and
// "mariadb" is a repository wearing a family's name, so it matched none of them.
// The generated docker-compose.yml simply had no `db` service, while the
// `dbdata` volume was still declared, so nothing about the file looked
// truncated.
//
// What made it expensive is that everything downstream kept saying yes.
// `madock start` reported success, `madock status` listed the other services
// without mentioning a database, and `madock info` printed "Database: type
// MARIADB, host db". On Magento the failure finally surfaced as `bin/magento`
// answering "There are no commands defined in the … namespace", which points
// nowhere near the cause.
//
// Found on a project whose config madock itself wrote in March 2024. The value
// is set here with `config:set` because today's setup no longer writes it —
// which is exactly the shape of the problem: the configurations at risk are the
// ones already on disk, and a golden test written from today's writer would
// never produce one.
func TestMariaDbTypeStartsADatabase(t *testing.T) {
	p := newProject(t, "e2emariadb")

	p.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2emariadb.test",
	)

	p.run(1*time.Minute, "config:set", "--name", "db/type", "--value", "MariaDB")

	// Checked rather than assumed: `config:set` writes only when
	// `configs.IsOption` recognises the name, and does nothing — successfully,
	// exit zero — when it does not. A test that skipped this step would set up
	// the wrong project and then pass, which is how this one first read green
	// against a binary that still had the bug.
	requireContains(t, p.run(1*time.Minute, "config:list"), "MariaDB", "db/type after config:set")

	p.run(20*time.Minute, "start")

	// The compose file itself, before asking docker anything. This is where the
	// bug was: the `db` service was absent from the generated file while the
	// `dbdata` volume was still declared, so nothing about the output looked
	// truncated.
	// Reported rather than fatal, so the status check below still runs: the two
	// answers are worth having together. The first version of this test asked
	// only `status --json` and passed against a binary that still had the bug,
	// which is the whole reason the file is read here.
	generated := readFile(t, p.generated("docker-compose.yml"))
	if !strings.Contains(generated, "\n  db:") {
		t.Errorf("db/type=MariaDB generated a compose file with no db service:\n%s", generated)
	}

	var payload struct {
		Data struct {
			Services []struct {
				Service string `json:"service"`
				Running bool   `json:"running"`
			} `json:"services"`
		} `json:"data"`
	}
	decode(t, p.run(3*time.Minute, "status", "--json"), &payload, "status --json")

	found := false
	for _, service := range payload.Data.Services {
		if service.Service != "db" {
			continue
		}
		found = true
		if !service.Running {
			t.Errorf("the db service exists but is not running: %+v", payload.Data.Services)
		}
	}
	if !found {
		// The old failure, stated as the assertion: not "the database is
		// broken" but "there is no database service at all", which is what
		// `status` was quietly reporting.
		t.Errorf("db/type=MariaDB produced a project with no db service: %+v", payload.Data.Services)
	}
}
