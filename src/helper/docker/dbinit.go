package docker

import (
	"os/exec"
	"strings"
	"time"

	"github.com/faradey/madock/v4/src/helper/cli/fmtc"
	"github.com/faradey/madock/v4/src/helper/configs"
)

// A database image initialises its data directory on the first start: it
// brings up a temporary server, runs the account and schema SQL against it,
// stops it, and only then starts for real. That sequence is not transactional.
// A container recreated in the middle of it — `setup`, a `config:set`, and a
// `start` within the same ten seconds, which is what every test does and what
// a person does on a new server — leaves a data directory that has the `mysql`
// schema, so the next start skips initialisation, and has whichever accounts
// the script had reached. Measured in CI on 2026-09-11: root@localhost with a
// password, no root@'%', and the project's own `db:execute` answering
// `ERROR 1130 (HY000): Host '172.20.0.4' is not allowed to connect` for as long
// as the volume lives. That is the "MariaDB directory already full on a fresh
// project" that was struck off the backlog as unreproducible.
//
// The fix is to not do that: before a database container is stopped or
// recreated, wait for its initialisation to finish. The entrypoints say so in
// their own logs, and those are the markers matched below.

// dbServices are the services whose images initialise a data directory.
var dbServices = []string{"db", "db2"}

// dbInitTimeout bounds the wait. Initialisation is ten to thirty seconds on a
// laptop; a container stuck longer than this is not initialising, it is
// broken, and holding the command hostage to it helps nobody.
const dbInitTimeout = 3 * time.Minute

// initInProgress reads a database container's log and reports whether the
// image is still between "started initialising" and "done".
//
// MariaDB and MySQL print "Temporary server started" and later
// "init process done"; PostgreSQL prints "init process complete". A log with
// none of the start markers is a container that found its directory already
// initialised, which is the common case and not a wait.
func initInProgress(log string) bool {
	started := strings.Contains(log, "Temporary server started") ||
		strings.Contains(log, "Initializing database files") ||
		strings.Contains(log, "initializing database")
	if !started {
		return false
	}

	done := strings.Contains(log, "init process done") ||
		strings.Contains(log, "init process complete") ||
		strings.Contains(log, "Temporary server stopped")

	return !done
}

// containerLog is replaceable so the wait can be exercised without a daemon.
var containerLog = func(container string) string {
	out, err := exec.Command("docker", "logs", "--tail", "300", container).CombinedOutput()
	if err != nil {
		return ""
	}

	return string(out)
}

// WaitForDatabaseInit blocks until no database container of the project is
// mid-initialisation, or the timeout passes.
//
// Called before anything that stops or recreates the project's containers. A
// container that does not exist, or whose log cannot be read, is not waited
// for: the point is to avoid interrupting work in progress, not to invent it.
func WaitForDatabaseInit(projectName string) {
	projectConf := configs.GetProjectConfig(projectName)
	deadline := time.Now().Add(dbInitTimeout)
	said := false

	for _, service := range dbServices {
		if service == "db2" && projectConf["db2/enabled"] != "true" {
			continue
		}
		container := GetContainerName(projectConf, projectName, service)

		for initInProgress(containerLog(container)) {
			if !said {
				fmtc.ToDoLn("Waiting for " + service + " to finish initialising its data directory before touching it")
				said = true
			}
			if time.Now().After(deadline) {
				fmtc.WarningLn(service + " is still initialising after " + dbInitTimeout.String() + " — going ahead; check its log if the database refuses connections afterwards")

				return
			}
			time.Sleep(2 * time.Second)
		}
	}
}
