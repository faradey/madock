//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestServiceEnabledAtTheAskedVersion covers `--service-version`, the flag that
// decides what a service is *made of* rather than whether it exists.
//
// Only four services take a version at enable time, and without the flag madock
// asks interactively — so on a machine with no terminal the flag is not a
// convenience, it is the only way in. That makes two things worth pinning: the
// version reaches the generated stack, and asking for one does not leave the
// command waiting for an answer nobody can give.
//
// The version asked for is deliberately not the default. Otherwise a flag that
// is parsed and thrown away passes.
func TestServiceEnabledAtTheAskedVersion(t *testing.T) {
	p := newProject(t, "e2esvcver")

	p.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2esvcver.test",
	)
	p.run(20*time.Minute, "start")

	// 8.0.4 against the 8.1.3 in the defaults.
	p.run(10*time.Minute, "service:enable", "valkey", "--service-version", "8.0.4")

	requireContains(t, p.run(2*time.Minute, "config:list"), "8.0.4",
		"the version the flag asked for, in the project config")

	// The generated compose file is what docker is handed, so it is where a
	// version that was stored and then ignored would show up.
	compose, err := os.ReadFile(p.generated("docker-compose.yml"))
	if err != nil {
		t.Fatalf("reading the generated compose file: %v", err)
	}
	if !strings.Contains(string(compose), "valkey/valkey:8.0.4") {
		t.Errorf("the generated stack does not run the version that was asked for:\n%s",
			serviceBlock(string(compose), "valkey"))
	}

	// `valkeydb` is what the service is called in the stack, not `valkey` —
	// the name the flag takes and the name status prints are not the same word.
	requireContains(t, p.run(3*time.Minute, "status"), "valkeydb running",
		"the service the flag enabled")
}

// TestStartWithChownTakesOwnershipBack covers `--with-chown`, which exists for
// one situation and is invisible until you are in it.
//
// Anything a container writes as root — a cache directory, a composer install
// that was run with the wrong user, a file created by an entrypoint — belongs to
// root on the host too, and cannot be edited, moved or deleted from outside.
// `--with-chown` hands the project directory back to whoever ran madock.
//
// The test creates the situation rather than hoping for it: a file written as
// root inside the container, checked from outside to be sure it really is
// root-owned, and only then the flag.
func TestStartWithChownTakesOwnershipBack(t *testing.T) {
	p := newProject(t, "e2echown")

	p.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2echown.test",
	)
	p.run(20*time.Minute, "start")

	p.runWithInput(2*time.Minute, "touch /var/www/html/root-owned\n", "bash", "-u", "root")

	written := filepath.Join(p.runDir, "root-owned")
	owner, err := fileOwner(written)
	if err != nil {
		t.Fatalf("the file created inside the container did not appear outside: %v", err)
	}
	if owner != 0 {
		t.Skipf("the file came out owned by uid %d rather than root, so this test has nothing to fix", owner)
	}

	p.run(20*time.Minute, "start", "--with-chown")

	owner, err = fileOwner(written)
	if err != nil {
		t.Fatalf("the file disappeared during start --with-chown: %v", err)
	}
	if owner != os.Getuid() {
		t.Errorf("start --with-chown left the file owned by uid %d, not %d", owner, os.Getuid())
	}
}

// TestRebuildForceKeepsTheData covers `rebuild --force`, which differs from a
// plain rebuild in one word: the containers are killed rather than stopped.
//
// That is what makes it the command for a stack that will not come down
// politely, and also what makes it worth a test — a database killed mid-write
// has to come back through crash recovery with its rows intact. Nothing else in
// the suite kills a database on purpose.
func TestRebuildForceKeepsTheData(t *testing.T) {
	p := newProject(t, "e2erebuildf")

	p.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2erebuildf.test",
	)
	p.run(20*time.Minute, "start")

	p.freshTable("probe", "(note VARCHAR(32))")
	p.query("INSERT INTO probe VALUES ('written-before-rebuild')")

	p.run(20*time.Minute, "rebuild", "--force")

	requireContains(t, p.query("SELECT note FROM probe"), "written-before-rebuild",
		"the row a killed database has to bring back")
	requireContains(t, p.run(3*time.Minute, "status"), "db running",
		"the database after a forced rebuild")
}

// TestExportAsAnotherContainerUser covers `db:export -u`, which chooses the user
// the dump command runs as *inside* the container — not a database login, which
// is the confusion the flag invites.
//
// The default is `mysql`; root is what somebody reaches for when a permission
// gets in the way, and nothing has ever run it.
func TestExportAsAnotherContainerUser(t *testing.T) {
	p := newProject(t, "e2eexportu")

	p.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2eexportu.test",
	)
	p.run(20*time.Minute, "start")

	p.freshTable("probe", "(note VARCHAR(32))")
	p.query("INSERT INTO probe VALUES ('in-the-dump')")

	dump := dumpContents(t, p, p.run(10*time.Minute, "db:export", "-n", "asroot", "-u", "root", "--json"))
	requireContains(t, dump, "probe", "the table in a dump taken as root")
	requireContains(t, dump, "in-the-dump", "the row in a dump taken as root")
}

// fileOwner returns the uid owning a file.
func fileOwner(path string) (int, error) {
	info, err := os.Stat(path)
	if err != nil {
		return -1, err
	}
	return int(info.Sys().(*syscall.Stat_t).Uid), nil
}

// serviceBlock returns the part of a compose file around a service name, for
// failure messages: printing the whole generated stack buries the one line that
// matters.
func serviceBlock(compose, service string) string {
	idx := strings.Index(compose, service)
	if idx < 0 {
		return "no mention of " + service + " in the generated compose file"
	}
	end := idx + 400
	if end > len(compose) {
		end = len(compose)
	}
	return compose[idx:end]
}
