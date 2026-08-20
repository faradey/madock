package project

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/faradey/madock/v4/src/helper/testenv"
)

// TestMariaDbTypeStillRendersADatabaseService is the bug end to end, at the
// level it was visible.
//
// Every compose snippet gates on one of three engine families —
// db.yml on "mysql", db-postgresql.yml on "postgresql", db-mongodb.yml on
// "mongodb" — and a project whose config said `MariaDB` matched none of them.
// The generated docker-compose.yml came out with no `db` service at all, while
// the `dbdata` volume was still declared (that gate is unconditional), so
// nothing in the output looked truncated. `madock start` reported success,
// `madock status` listed php and nginx and said nothing about a database, and
// `madock info` printed "type MARIADB, host db". On Magento the failure
// eventually surfaced as `bin/magento` answering "There are no commands defined
// in the ... namespace", which points nowhere near the cause.
//
// Found on a project whose config madock itself wrote in March 2024, so the
// radius is every configuration old enough to predate the current writer, plus
// every hand-edited .madock/config.xml.
func TestMariaDbTypeStillRendersADatabaseService(t *testing.T) {
	for _, dbType := range []string{"MariaDB", "mariadb", "mysql"} {
		t.Run(dbType, func(t *testing.T) {
			projectName := "dbtype"
			env := testenv.SetupWith(t, projectName, "dbtype.test", map[string]string{
				"db/enabled":    "true",
				"db/type":       dbType,
				"db/repository": "mariadb",
				"db/version":    "10.6",
			})

			MakeConf(projectName)

			rendered := testenv.Collect(t, filepath.Join(env.ExecDir, "aruntime", "projects", projectName), env)

			compose := ""
			for name, body := range rendered {
				if strings.HasSuffix(name, "docker-compose.yml") {
					compose = body
					break
				}
			}
			if compose == "" {
				t.Fatal("MakeConf produced no docker-compose.yml")
			}

			// The service block, not the volume: `dbdata` is declared
			// unconditionally and was present throughout the bug, which is part
			// of why the output did not look wrong.
			if !strings.Contains(compose, "\n  db:") {
				t.Errorf("db/type=%q produced a compose file with no db service:\n%s", dbType, compose)
			}
		})
	}
}
