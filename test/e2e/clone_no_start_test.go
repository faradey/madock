//go:build e2e

package e2e

import (
	"os/exec"
	"strings"
	"testing"
	"time"
)

// containersOf lists the containers of a project by name, leaving out the
// snapshot helper: it is created by every clone and stopped, and is not part of
// the stack a clone starts.
func containersOf(t *testing.T, project string) []string {
	t.Helper()
	out, err := exec.Command("docker", "ps", "-a", "--format", "{{.Names}}", "--filter", "name="+project).Output()
	if err != nil {
		t.Fatalf("docker ps: %v", err)
	}
	var names []string
	for _, name := range strings.Fields(string(out)) {
		if strings.Contains(name, "-snapshot-") {
			continue
		}
		names = append(names, name)
	}
	return names
}

// TestCloneWithNoStartCreatesNoContainerUntilStart — `project:clone --no-start`
// fills the clone's volumes from the snapshot and creates no container of the
// clone. `madock start` creates them, and the clone holds the snapshot's state.
func TestCloneWithNoStartCreatesNoContainerUntilStart(t *testing.T) {
	install := newInstallation(t)
	source := install.project("e2enostart")
	clone := install.project("e2enostartcopy")

	source.run(setupTimeout, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2enostart.test",
	)
	source.run(20*time.Minute, "start")

	source.freshTable("state", "(note VARCHAR(32))")
	source.query("INSERT INTO state VALUES ('saved')")
	source.run(10*time.Minute, "snapshot:create", "-n", "golden")
	source.query("INSERT INTO state VALUES ('written-after')")

	source.run(30*time.Minute, "project:clone", "-n", clone.name, "-s=-copy", "--from", "golden", "--no-start")

	if names := containersOf(t, clone.name); len(names) > 0 {
		t.Errorf("project:clone --no-start created containers: %v", names)
	}

	clone.run(20*time.Minute, "start")
	if names := containersOf(t, clone.name); len(names) == 0 {
		t.Errorf("madock start after --no-start created no containers")
	}

	got := clone.query("SELECT note FROM state")
	requireContains(t, got, "saved", "the clone should hold the snapshot's row")
	if strings.Contains(got, "written-after") {
		t.Errorf("the clone holds a row written after the snapshot:\n%s", got)
	}
}

// TestCloneWithNoStartAndAPersistedBrokerStartsTheWholeStack — the muted clone
// of a project with a persisted RabbitMQ empties the broker's queues by starting
// the broker alone. With --no-start that container must go too, or `madock
// start` finds one container of the stack, wakes only it, and reports success
// with the database and the rest never created.
func TestCloneWithNoStartAndAPersistedBrokerStartsTheWholeStack(t *testing.T) {
	install := newInstallation(t)
	source := install.project("e2enostartrmq")
	clone := install.project("e2enostartrmqcopy")

	source.run(setupTimeout, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2enostartrmq.test",
	)
	source.run(2*time.Minute, "config:set", "-n", "rabbitmq/enabled", "-v", "true")
	source.run(2*time.Minute, "config:set", "-n", "rabbitmq/persist", "-v", "true")
	source.run(25*time.Minute, "start")

	source.freshTable("state", "(note VARCHAR(32))")
	source.query("INSERT INTO state VALUES ('saved')")
	source.run(10*time.Minute, "snapshot:create", "-n", "golden")

	source.run(30*time.Minute, "project:clone", "-n", clone.name, "-s=-copy", "--from", "golden", "--muted", "--no-start")

	if names := containersOf(t, clone.name); len(names) > 0 {
		t.Errorf("project:clone --no-start left containers behind: %v", names)
	}

	clone.run(25*time.Minute, "start")
	names := strings.Join(containersOf(t, clone.name), " ")
	for _, service := range []string{"-db-1", "-rabbitmq-1"} {
		if !strings.Contains(names, service) {
			t.Errorf("after madock start the clone has no %s container; containers: %q", service, names)
		}
	}

	requireContains(t, clone.query("SELECT note FROM state"), "saved", "the clone should hold the snapshot's row")
}

// TestCloneLeavesNoHelperAndNoNetworkBehind — the snapshot helper goes once the
// clone's volumes are filled, and with --no-start so does the network it was
// created on. A stopped helper was reported by `start` as "Left over from a
// service this project no longer has: snapshot", and each unstarted clone held
// an address pool: the fourth clone of a test run failed with "all predefined
// address pools have been fully subnetted".
func TestCloneLeavesNoHelperAndNoNetworkBehind(t *testing.T) {
	install := newInstallation(t)
	source := install.project("e2ehelper")
	clone := install.project("e2ehelpercopy")

	source.run(setupTimeout, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2ehelper.test",
	)
	source.run(20*time.Minute, "start")
	source.run(10*time.Minute, "snapshot:create", "-n", "golden")

	source.run(30*time.Minute, "project:clone", "-n", clone.name, "-s=-copy", "--from", "golden", "--no-start")

	out, err := exec.Command("docker", "ps", "-a", "--format", "{{.Names}}", "--filter", "label=com.docker.compose.project=madock_"+clone.name).Output()
	if err != nil {
		t.Fatalf("docker ps: %v", err)
	}
	if names := strings.Fields(string(out)); len(names) > 0 {
		t.Errorf("the clone left containers behind: %v", names)
	}
	nets, err := exec.Command("docker", "network", "ls", "--format", "{{.Name}}", "--filter", "label=com.docker.compose.project=madock_"+clone.name).Output()
	if err != nil {
		t.Fatalf("docker network ls: %v", err)
	}
	if names := strings.Fields(string(nets)); len(names) > 0 {
		t.Errorf("the unstarted clone holds networks: %v", names)
	}

	started := clone.run(20*time.Minute, "start")
	if strings.Contains(started, "Left over") {
		t.Errorf("start reported a leftover on a fresh clone:\n%s", started)
	}
}
