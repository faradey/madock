package status

import (
	"os/exec"
	"strings"

	"github.com/faradey/madock/v4/src/helper/paths"
)

// markOrphans flags containers that belong to this compose project but are not
// among the services its files declare.
//
// `docker compose ps` lists by project, not by file: the project name comes from
// the directory the compose file sits in, so a container created from an earlier
// version of that file keeps being reported after the service is gone from it.
// `status` presented those as ordinary services, which is a wrong answer rather
// than a missing one — and it hid a real defect for a day. A project whose config
// said `db/type: MariaDB` generated a compose file with **no db service at all**,
// and `status --json` went on listing `db` as running, so a test written against
// `status` passed against the broken build. Only reading the generated file
// showed the truth.
//
// Reported, never removed. `--remove-orphans` would delete them, and that is the
// wrong trade here: the compose file is generated from a config, so a rendering
// bug decides what counts as an orphan — and the very bug above would then have
// aimed the deletion at a running database container. Naming the container costs
// nothing and cannot destroy anything; the person reading it can decide.
//
// `known` false means the question could not be asked, and then nothing is
// flagged: an unanswered check must not turn into "everything here is an orphan".
func markOrphans(services []ServiceStatus, known map[string]bool, ok bool) []ServiceStatus {
	if !ok || len(services) == 0 {
		return services
	}

	for i := range services {
		// A container compose could not name a service for is not evidence of
		// anything — it is compose declining to answer, and the whole point of
		// this is to stop turning silence into a claim.
		if services[i].Service == "" {
			continue
		}
		if !known[services[i].Service] {
			services[i].Orphan = true
		}
	}

	return services
}

// definedServices asks compose which services a stack's files declare.
//
// Asked of docker rather than read out of the YAML, because the answer has to be
// the same one `up` acts on: the override file, profiles and any interpolation
// are compose's to resolve, and a second implementation here would disagree with
// it on exactly the projects where it matters. The override is included when it
// exists for the same reason — `up` passes both, so a service declared only there
// is not an orphan.
//
// The second return value is whether the question was answered at all.
func definedServices(composePath, overridePath string) (map[string]bool, bool) {
	if !paths.IsFileExist(composePath) {
		return nil, false
	}

	args := []string{"compose", "-f", composePath}
	if overridePath != "" && paths.IsFileExist(overridePath) {
		args = append(args, "-f", overridePath)
	}
	args = append(args, "config", "--services")

	cmd := exec.Command("docker", args...)
	// stdout only: compose writes its deprecation warnings to stderr, and folding
	// them in here would turn a warning into a service name.
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, false
	}

	known := map[string]bool{}
	for _, line := range strings.Split(string(out), "\n") {
		if name := strings.TrimSpace(line); name != "" {
			known[name] = true
		}
	}

	// An empty list is not an answer worth acting on. A stack always declares
	// something, so nothing coming back means compose printed to a place this did
	// not read, and flagging every running container as an orphan on the strength
	// of that would be the loudest possible way to be wrong.
	if len(known) == 0 {
		return nil, false
	}

	return known, true
}
