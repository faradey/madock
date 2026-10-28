//go:build e2e

package e2e

import (
	"strings"
	"testing"
	"time"
)

// TestAMutedCloneWithAPersistedBrokerStartsTheWholeStack — the purge of a muted
// clone's queues starts the broker alone. Its container must not be left
// behind stopped: `start` would then find a container of the stack, wake only
// it when the stack record matches, and report success with the database and
// the rest never created.
func TestAMutedCloneWithAPersistedBrokerStartsTheWholeStack(t *testing.T) {
	install := newInstallation(t)
	source := install.project("e2emutestack")
	clone := install.project("e2emutestackcopy")

	source.run(setupTimeout, "setup", "-y", "--platform=custom", "--language=none", "--hosts=e2emutestack.test")
	source.run(2*time.Minute, "config:set", "-n", "rabbitmq/enabled", "-v", "true")
	source.run(2*time.Minute, "config:set", "-n", "rabbitmq/persist", "-v", "true")
	source.run(25*time.Minute, "start")
	source.freshTable("state", "(note VARCHAR(32))")
	source.query("INSERT INTO state VALUES ('saved')")
	source.run(10*time.Minute, "snapshot:create", "-n", "golden")

	source.run(30*time.Minute, "project:clone", "-n", clone.name, "-s=-copy", "--from", "golden", "--muted")
	clone.run(25*time.Minute, "start")

	names := strings.Join(containersOf(t, clone.name), " ")
	for _, service := range []string{"-db-1", "-rabbitmq-1"} {
		if !strings.Contains(names, service) {
			t.Errorf("after madock start the muted clone has no %s container; containers: %q", service, names)
		}
	}
	requireContains(t, clone.query("SELECT note FROM state"), "saved", "the clone should hold the snapshot's row")
}

// TestAMutedCloneCannotActAsItsSource — a clone has the source's database and,
// with rabbitmq/persist, the source's queued messages. A muted one must not
// run them: its queues are emptied (the queues themselves stay, or the
// application's next publish is refused), its cron is off and cannot be
// switched on, and the source keeps everything it had.
func TestAMutedCloneCannotActAsItsSource(t *testing.T) {
	install := newInstallation(t)
	source := install.project("e2emute")

	source.run(setupTimeout, "setup", "-y", "--platform=custom", "--language=none", "--hosts=e2emute.test")
	source.run(2*time.Minute, "config:set", "-n", "rabbitmq/enabled", "-v", "true")
	source.run(2*time.Minute, "config:set", "-n", "rabbitmq/persist", "-v", "true")
	source.run(25*time.Minute, "start")

	// As the broker's user: rabbitmqctl needs the Erlang cookie, which the
	// project's user madock cli runs as cannot read — and root must not touch,
	// since a cookie root creates first is one the broker cannot read.
	rabbit := func(p *project, args ...string) string {
		out, err := p.tryRunWith(3*time.Minute, "", []string{"MADOCK_USER=rabbitmq"}, append([]string{"cli", "--service", "rabbitmq"}, args...)...)
		if err != nil {
			logs, _ := p.tryRun(2*time.Minute, "logs", "-s", "rabbitmq")
			if len(logs) > 4000 {
				logs = logs[len(logs)-4000:]
			}
			t.Fatalf("%v: %v\n%s\n--- rabbitmq log ---\n%s", args, err, out, logs)
		}
		return out
	}
	// Wait for the broker and its management API — rabbitmqadmin speaks HTTP,
	// and the plugin listens a while after the node answers a ping — then one
	// durable queue with one persistent message.
	rabbit(source, "bash", "-c", "for i in $(seq 1 90); do rabbitmq-diagnostics -q check_port_listener 15672 >/dev/null 2>&1 && exit 0; sleep 2; done; exit 1")
	rabbit(source, "rabbitmqadmin", "declare", "queue", "name=orders.export", "durable=true")
	rabbit(source, "rabbitmqadmin", "publish", "exchange=amq.default", "routing_key=orders.export", "payload=order-1", `properties={"delivery_mode":2}`)

	messages := func(p *project) string {
		out := rabbit(p, "rabbitmqctl", "-q", "--no-table-headers", "list_queues", "name", "messages")
		for _, line := range strings.Split(out, "\n") {
			if f := strings.Fields(line); len(f) == 2 && f[0] == "orders.export" {
				return f[1]
			}
		}
		return "missing"
	}
	if got := messages(source); got != "1" {
		t.Fatalf("the source's queue holds %s messages before the clone, want 1", got)
	}

	source.run(30*time.Minute, "project:clone", "-n", "e2emutecopy", "-s=-copy", "--muted")
	clone := install.project("e2emutecopy")
	clone.run(20*time.Minute, "start")
	rabbit(clone, "bash", "-c", "for i in $(seq 1 90); do rabbitmq-diagnostics -q check_running >/dev/null 2>&1 && exit 0; sleep 2; done; exit 1")

	if got := messages(clone); got != "0" {
		t.Errorf("the muted clone's queue holds %s messages, want the queue kept and empty", got)
	}
	if got := messages(source); got != "1" {
		t.Errorf("cloning emptied the source's queue: %s messages", got)
	}

	if got := clone.configValue("cron/enabled"); got != "false" {
		t.Errorf("the muted clone has cron/enabled %q", got)
	}
	if out, err := clone.tryRun(2*time.Minute, "cron:enable"); err == nil {
		t.Errorf("cron:enable succeeded on a muted clone:\n%s", out)
	}
}
