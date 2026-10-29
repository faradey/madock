package docker

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/faradey/madock/v4/src/helper/cli/attr"
	"github.com/faradey/madock/v4/src/helper/cli/fmtc"
)

// Every madock project is a compose project of its own, and compose gives each
// one a network. Docker carves those out of a fixed set of address pools — a
// few dozen networks on a default daemon — and a network that nothing uses any
// more still holds its subnet. Found on a machine running test clones of one
// reference project: 26 networks with no container on them, and the next clone
// failed with
//
//	failed to create network madock_<name>_default: Error response from daemon:
//	all predefined address pools have been fully subnetted
//
// which, under --quiet, reached the user as `exit status 1` and nothing else.

// poolsExhausted is docker's wording for "no subnet left for a new network".
const poolsExhausted = "all predefined address pools have been fully subnetted"

// runCompose runs a compose command whose failure must be explained, not only
// reported: docker's stderr goes where it always went — nowhere under --quiet —
// and is also kept, so the error carries docker's own words.
//
// A failure because the address pools ran out is answered once: madock removes
// the networks of its own that no container uses and runs the command again.
func runCompose(args ...string) error {
	err := runComposeOnce(args...)
	if err == nil || !strings.Contains(err.Error(), poolsExhausted) {
		return err
	}

	removed := PruneUnusedNetworks()
	if len(removed) == 0 {
		return fmt.Errorf("%w\nDocker has no address pool left for a new network, and madock has no unused network of its own to give back. Remove networks you no longer need: docker network ls", err)
	}
	fmtc.WarningLn(fmt.Sprintf("Docker ran out of address pools; removed %d unused madock network(s): %s", len(removed), strings.Join(removed, ", ")))

	return runComposeOnce(args...)
}

func runComposeOnce(args ...string) error {
	var stderr bytes.Buffer
	cmd := exec.Command("docker", args...)
	if attr.IsQuiet {
		cmd.Stderr = &stderr
	} else {
		cmd.Stdout = os.Stdout
		cmd.Stderr = io.MultiWriter(os.Stderr, &stderr)
	}

	if err := cmd.Run(); err != nil {
		if detail := lastLines(stderr.String(), 20); detail != "" {
			return fmt.Errorf("%w: %s", err, detail)
		}
		return err
	}
	return nil
}

// lastLines keeps the end of docker's output: the error is the last thing it
// prints, after the progress lines of everything that went well.
func lastLines(s string, n int) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

// PruneUnusedNetworks removes madock's networks that no container is attached
// to — running or stopped — and returns their names.
//
// Only madock's: a network counts when its compose project label starts with
// `madock_`, the same rule FindOrphans applies. Somebody's own compose stack on
// the same machine is not ours to tidy.
//
// A stopped container keeps its network: `docker network rm` would succeed,
// since a stopped container has no active endpoint, and the container would
// then refuse to start with "network not found". So the question asked is
// `ps -a --filter network=`, which counts the stopped ones too. A network
// removed here is recreated by the next `compose up` of its project.
func PruneUnusedNetworks() []string {
	var removed []string
	for _, line := range dockerLines("network", "ls") {
		name, label, ok := strings.Cut(strings.TrimSpace(line), "\t")
		if !ok || !isMadockProject(label) {
			continue
		}
		if removeNetworkIfUnused(name) {
			removed = append(removed, name)
		}
	}
	return removed
}

// removeProjectNetworksIfUnused gives back the networks of one project that no
// container uses any more — after its snapshot helper is gone, on a project
// whose own containers were never created.
func removeProjectNetworksIfUnused(projectName string) {
	for _, name := range dockerQuery("network", "ls", "--format", "{{.Name}}",
		"--filter", "label=com.docker.compose.project="+composeProjectName(projectName)) {
		removeNetworkIfUnused(name)
	}
}

func removeNetworkIfUnused(name string) bool {
	out, err := exec.Command("docker", "ps", "-aq", "--filter", "network="+name).Output()
	if err != nil || strings.TrimSpace(string(out)) != "" {
		return false
	}
	return exec.Command("docker", "network", "rm", name).Run() == nil
}

// isMadockProject reports whether a compose project label is one of madock's.
func isMadockProject(label string) bool {
	project, ours := strings.CutPrefix(label, "madock_")
	return ours && project != ""
}
