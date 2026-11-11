//go:build e2e

package e2e

import (
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestOrphansRemoveTakesOnlyAForgottenProject drives `project:orphans --remove`
// against a real daemon: what it removes, and the three things it must refuse —
// a registered project's label, a label a container still carries, and any of
// it on an installation that forbids destructive commands.
//
// Nothing is started. The leftovers are made with docker directly, carrying the
// label Compose would have written: a project's volume outliving its directory
// is the state the command is for, and starting a project only to delete its
// directory would spend twenty minutes reaching the same labels.
func TestOrphansRemoveTakesOnlyAForgottenProject(t *testing.T) {
	install := newInstallation(t)
	live := install.project("e2eorphlive")
	live.run(setupTimeout, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2eorphlive.test",
	)

	gone := labelledVolume(t, "madock_e2eorphgone")
	goneNet := labelledNetwork(t, "madock_e2eorphgone")
	liveData := labelledVolume(t, "madock_e2eorphlive")
	heldData := labelledVolume(t, "madock_e2eorphheld")
	labelledContainer(t, "madock_e2eorphheld")

	t.Cleanup(func() { writeInstallationConfig(t, live.execDir, "true") })

	// Forbidden: refused before anything is listed, with --force or without.
	writeInstallationConfig(t, live.execDir, "false")
	for _, args := range [][]string{
		{"project:orphans", "--remove", "e2eorphgone"},
		{"project:orphans", "--remove", "e2eorphgone", "--force"},
	} {
		out, err := live.tryRun(2*time.Minute, args...)
		requireStatusOne(t, err, out, strings.Join(args, " ")+" under allow_destructive_commands false")
		requireContains(t, out, "allow_destructive_commands", "the refusal should name the setting")
	}
	requireVolume(t, gone, true, "after a removal the installation forbids")

	writeInstallationConfig(t, live.execDir, "true")

	// A registered project's data is not a leftover, whatever the label says.
	out, err := live.tryRun(2*time.Minute, "project:orphans", "--remove", "e2eorphlive", "--force")
	requireStatusOne(t, err, out, "--remove of a registered project")
	requireContains(t, out, "e2eorphlive", "the refusal should name the registered project")
	requireVolume(t, liveData, true, "after --remove of the registered project's name")

	// A container under the label means somebody still runs, or ran, there.
	out, err = live.tryRun(2*time.Minute, "project:orphans", "--remove", "e2eorphheld", "--force")
	requireStatusOne(t, err, out, "--remove while a container carries the label")
	requireContains(t, out, "e2eorphheld-c", "the refusal should name the container")
	requireVolume(t, heldData, true, "after --remove refused for a container")

	// Without --force it only says what would go.
	out = live.run(2*time.Minute, "project:orphans", "--remove", "e2eorphgone")
	requireContains(t, out, gone, "the listing should name the volume")
	requireContains(t, out, goneNet, "the listing should name the network")
	requireVolume(t, gone, true, "after --remove without --force")

	live.run(2*time.Minute, "project:orphans", "--remove", "e2eorphgone", "--force")
	requireVolume(t, gone, false, "after --remove --force")
	if ids := dockerIDs(t, "network", "ls", "-q", "--filter", "name=^"+goneNet+"$"); len(ids) > 0 {
		t.Errorf("network %s survived --remove --force", goneNet)
	}
	requireVolume(t, liveData, true, "after another project's removal")
	requireVolume(t, heldData, true, "after another project's removal")
}

// labelledVolume creates a volume carrying the Compose label label, removed
// when the test ends if it is still there.
func labelledVolume(t *testing.T, label string) string {
	t.Helper()
	name := label + "_dbdata"
	dockerMust(t, "volume", "create", "--label", "com.docker.compose.project="+label, name)
	t.Cleanup(func() { _ = exec.Command("docker", "volume", "rm", "-f", name).Run() })
	return name
}

func labelledNetwork(t *testing.T, label string) string {
	t.Helper()
	name := label + "_default"
	dockerMust(t, "network", "create", "--label", "com.docker.compose.project="+label, name)
	t.Cleanup(func() { _ = exec.Command("docker", "network", "rm", name).Run() })
	return name
}

// labelledContainer creates, and does not start, a container carrying label:
// a stopped container counts as much as a running one.
func labelledContainer(t *testing.T, label string) {
	t.Helper()
	name := strings.TrimPrefix(label, "madock_") + "-c"
	dockerMust(t, "create", "--name", name, "--label", "com.docker.compose.project="+label, "alpine:3", "true")
	t.Cleanup(func() { _ = exec.Command("docker", "rm", "-f", name).Run() })
}

func dockerMust(t *testing.T, args ...string) {
	t.Helper()
	if out, err := exec.Command("docker", args...).CombinedOutput(); err != nil {
		t.Fatalf("docker %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func requireVolume(t *testing.T, name string, want bool, when string) {
	t.Helper()
	has := len(dockerIDs(t, "volume", "ls", "-q", "--filter", "name=^"+name+"$")) > 0
	if has != want {
		t.Errorf("volume %s present=%v %s, want %v", name, has, when, want)
	}
}
