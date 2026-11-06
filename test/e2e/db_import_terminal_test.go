//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestDbImportWithoutATerminalAnswersNoPromptItself runs what a script runs, with
// nothing on stdin. A failing statement stops the import with a failing status;
// only a flag asks for it to be skipped.
func TestDbImportWithoutATerminalAnswersNoPromptItself(t *testing.T) {
	p := newProject(t, "e2eimport")
	p.run(setupTimeout, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--db-type=mariadb",
		"--hosts=e2eimport.test",
	)
	p.run(20*time.Minute, "start")
	p.query("SELECT 1")

	dump := func(name, body string) string {
		t.Helper()
		path := filepath.Join(p.runDir, name)
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}
	table := "DROP TABLE IF EXISTS t;\nCREATE TABLE t (id INT PRIMARY KEY);\n"

	duplicate := dump("duplicate.sql", table+"INSERT INTO t VALUES (1);\nINSERT INTO t VALUES (1);\nINSERT INTO t VALUES (2);\n")
	out, err := p.tryRun(5*time.Minute, "db:import", duplicate)
	if err == nil {
		t.Errorf("a duplicate row exited 0:\n%s", out)
	}
	requireContains(t, out, "--force", "the refusal names the flag")
	if strings.Contains(out, "completed successfully") {
		t.Errorf("a duplicate row reported success:\n%s", out)
	}
	if rows := p.query("SELECT id FROM t"); strings.Contains(rows, "2") {
		t.Errorf("the statements after the duplicate ran: %s", rows)
	}

	p.run(5*time.Minute, "db:import", "--force", duplicate)
	if rows := p.query("SELECT id FROM t"); !strings.Contains(rows, "2") {
		t.Errorf("--force did not skip the duplicate: %s", rows)
	}

	// MariaDB has no GTID_PURGED, so this statement simply fails and the GTID
	// refusal itself is left to the unit test. What is checked is the flag.
	gtid := dump("gtid.sql", table+"SET @@GLOBAL.GTID_PURGED='3E11FA47-71CA-11E1-9E33-C80AA9429562:1-5';\nINSERT INTO t VALUES (7);\n")
	if out, err := p.tryRun(5*time.Minute, "db:import", gtid); err == nil {
		t.Errorf("a GTID statement exited 0 with no flag:\n%s", out)
	}
	p.run(5*time.Minute, "db:import", "--strip-gtid", gtid)
	if rows := p.query("SELECT id FROM t"); !strings.Contains(rows, "7") {
		t.Errorf("--strip-gtid did not import the rest of the dump: %s", rows)
	}
}
