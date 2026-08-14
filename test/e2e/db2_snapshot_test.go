//go:build e2e

package e2e

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestSnapshotAndInfoKnowAboutTheSecondDatabase covers the two commands that
// have to handle db2 without ever being told about it.
//
// `--service db2` is how the db commands are pointed at the second server, and
// the ones here have no such flag: `snapshot:create` decides for itself whether
// to archive a second data directory, and `db:info` decides for itself whether
// to describe a second database. Both read `db2/enabled` and neither is checked
// by anything.
//
// The snapshot half is the expensive one to get wrong, and it is silent in the
// worst way: a restore that rolls the first database back to the snapshot and
// leaves the second where it was does not fail, does not warn, and produces two
// databases that disagree about what happened — discovered by whoever needed the
// restore, which is the audience with the least patience for it.
func TestSnapshotAndInfoKnowAboutTheSecondDatabase(t *testing.T) {
	p := newProject(t, "e2edb2snap")

	p.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2edb2snap.test",
	)

	p.run(2*time.Minute, "config:set", "-n", "db2/enabled", "-v", "true")
	p.run(2*time.Minute, "config:set", "-n", "db2/database", "-v", "second")
	p.run(2*time.Minute, "config:set", "-n", "db2/user", "-v", "second_user")
	p.run(2*time.Minute, "config:set", "-n", "db2/password", "-v", "second_pw")
	p.run(2*time.Minute, "config:set", "-n", "db2/root_password", "-v", "second_root")

	// Before anything runs: describing a project must not change it. `info`
	// carries that rule in a comment and honours it; `db:info` reads the same
	// registry, and a project that has never started has no ports to report.
	registry := filepath.Join(p.execDir, "aruntime", "ports.conf")
	before := readIfExists(t, registry)
	p.run(2*time.Minute, "db:info")
	if after := readIfExists(t, registry); after != before {
		t.Errorf("db:info allocated a port for a project that has never started.\nbefore:\n%s\nafter:\n%s", before, after)
	}

	p.run(25*time.Minute, "start")
	waitForSecondDatabase(t, p)

	// Both databases described, each with its own credentials. Reporting db's
	// name for db2 is the failure that makes somebody connect to the wrong
	// server and believe the data is missing.
	var described struct {
		Data struct {
			Databases []struct {
				Name     string `json:"name"`
				Database string `json:"database"`
				User     string `json:"user"`
			} `json:"databases"`
		} `json:"data"`
	}
	out := p.run(2*time.Minute, "db:info", "--json")
	if err := json.Unmarshal([]byte(jsonPart(out)), &described); err != nil {
		t.Fatalf("db:info --json did not decode: %v\n%s", err, out)
	}
	if len(described.Data.Databases) != 2 {
		t.Fatalf("db:info described %d databases, expected two:\n%s", len(described.Data.Databases), out)
	}
	second := described.Data.Databases[1]
	if second.Database != "second" || second.User != "second_user" {
		t.Errorf("db:info described the second database as %q/%q, not second/second_user:\n%s",
			second.Database, second.User, out)
	}

	// A row in each, distinguishable, so a restore that only reaches one of them
	// says which one.
	p.freshTable("first_probe", "(note VARCHAR(32))")
	p.query("INSERT INTO first_probe VALUES ('before-snapshot')")
	p.querySecond("DROP TABLE IF EXISTS second_probe")
	p.querySecond("CREATE TABLE second_probe (note VARCHAR(32))")
	p.querySecond("INSERT INTO second_probe VALUES ('before-snapshot')")

	p.run(15*time.Minute, "snapshot:create", "-n", "both-databases")

	// The archive itself, because its absence is the mechanism behind a restore
	// that silently leaves db2 alone: restore only unpacks db2.tar.gz if it is
	// there. The directory is `snapshot-<name>-<timestamp>`, so it is found by
	// prefix rather than by the name that was asked for.
	snapshotDir := snapshotDirNamed(t, p, "both-databases")
	requireFile(t, filepath.Join(snapshotDir, "db2.tar.gz"),
		"the second database's archive inside the snapshot")
	requireFile(t, filepath.Join(snapshotDir, "db.tar.gz"),
		"the first database's archive inside the snapshot")

	// snapshot:create takes the databases down to copy them, and a container that
	// is back is not a server accepting connections yet. Without this wait the very
	// next write to db2 answered `ERROR 2002 (HY000): Can't connect to server on
	// 'db2'` — on a busy runner, and only there: the same commit passed in a second
	// run that happened to be thirty seconds slower. The first database is waited
	// for the same way at the top of the test; this is the second place that needs
	// it and did not have it.
	waitForSecondDatabase(t, p)

	p.query("INSERT INTO first_probe VALUES ('after-snapshot')")
	p.querySecond("INSERT INTO second_probe VALUES ('after-snapshot')")

	p.run(20*time.Minute, "snapshot:restore", "--name", "both-databases")

	// Same reason as above: restore also stops the databases to replace their data
	// directories, so the assertions below have to wait for db2 to answer rather
	// than assume it does.
	waitForSecondDatabase(t, p)

	first := p.query("SELECT note FROM first_probe")
	requireContains(t, first, "before-snapshot", "the first database after the restore")
	if strings.Contains(first, "after-snapshot") {
		t.Errorf("the first database was not rolled back:\n%s", first)
	}

	restoredSecond := p.querySecond("SELECT note FROM second_probe")
	requireContains(t, restoredSecond, "before-snapshot", "the second database after the restore")
	if strings.Contains(restoredSecond, "after-snapshot") {
		t.Errorf("the snapshot restored the first database and left the second one where it was:\n%s", restoredSecond)
	}
}

// snapshotDirNamed returns the snapshot directory created for a given -n name.
//
// snapshot:create appends a timestamp, so the directory is
// `snapshot-<name>-<when>` and the caller cannot know it. Matching by prefix is
// the only way; more than one match means an earlier run left something behind
// and the test would otherwise pick one at random.
func snapshotDirNamed(t *testing.T, p *project, name string) string {
	t.Helper()

	root := filepath.Join(p.execDir, "projects", p.name, "backup", "snapshot")
	matches, err := filepath.Glob(filepath.Join(root, "snapshot-"+name+"-*"))
	if err != nil {
		t.Fatalf("looking for the snapshot in %s: %v", root, err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected one snapshot named %q in %s, found %v", name, root, matches)
	}
	return matches[0]
}
