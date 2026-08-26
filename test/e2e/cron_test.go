//go:build e2e

package e2e

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestCronReportsWhatTheContainerHas covers the toggle and, more importantly,
// what `status` says about it afterwards.
//
// Enabling cron is a command run inside the container, and it can fail. The
// configuration flag would still read true either way, so answering from the
// flag means reporting a scheduler that may not be there — and everything that
// runs on a schedule then quietly does not. Nobody notices until an order
// confirmation or a reindex is missing.
//
// So the assertions are about the container's answer, not the setting's.
func TestCronReportsWhatTheContainerHas(t *testing.T) {
	p := newProject(t, "e2ecron")

	p.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2ecron.test",
	)
	p.run(20*time.Minute, "start")

	requireContains(t, p.run(3*time.Minute, "status"), "Cron is not running", "cron before enabling it")
	if running, enabled := cronFromJSON(t, p); running || enabled {
		t.Errorf("cron reports running=%v enabled=%v before it was enabled", running, enabled)
	}

	p.run(5*time.Minute, "cron:enable")

	status := p.run(3*time.Minute, "status")
	requireContains(t, status, "Cron is running", "cron after enabling it")
	if strings.Contains(status, "enabled but not running") {
		t.Errorf("cron was asked for and did not start:\n%s", status)
	}

	running, enabled := cronFromJSON(t, p)
	if !running || !enabled {
		t.Errorf("after cron:enable the report is running=%v enabled=%v; both should be true", running, enabled)
	}

	p.run(5*time.Minute, "cron:disable")

	requireContains(t, p.run(3*time.Minute, "status"), "Cron is not running", "cron after disabling it")
	if running, _ := cronFromJSON(t, p); running {
		t.Error("cron is still running after being disabled")
	}
}

// TestCronSurvivesARestartOfItsContainer is the whole of the first defect,
// stated as a measurement.
//
// No application image starts cron: the php container's CMD is php-fpm and the
// Node one's is the dev server. The daemon exists only because madock ran a
// command inside a container that was already up, so restarting that container
// takes it away — and the crontab, which lives in the container's filesystem,
// survives. The project then looks exactly as it did: jobs installed, nothing
// running them.
//
// It is `service:restart` and not `restart` deliberately: that is the command a
// finished deploy runs, and on a Node project on 2026-08-26 it had left the
// production scheduler dead for six hours before anybody looked.
func TestCronSurvivesARestartOfItsContainer(t *testing.T) {
	p := newProject(t, "e2ecronrestart")

	p.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2ecronrestart.test",
	)
	installCronJob(t, p, "* * * * * /bin/true")
	p.run(20*time.Minute, "start")
	p.run(5*time.Minute, "cron:enable")

	if running, _ := cronFromJSON(t, p); !running {
		t.Fatal("cron did not start, so the restart proves nothing")
	}

	p.run(5*time.Minute, "service:restart", "app")

	running, _ := cronFromJSON(t, p)
	if !running {
		t.Error("restarting the application's container left cron dead — a deploy does exactly this")
	}
	if got := cronProcesses(t, p, "app"); got == "" {
		t.Error("no cron process in the container after the restart")
	}

	// The jobs have to come back with the daemon. A cron with an empty crontab
	// is the same silence wearing a healthy status.
	out, err := p.tryRun(3*time.Minute, "cron:status")
	if err != nil {
		t.Errorf("cron:status reports a problem after the restart: %v\n%s", err, out)
	}
	requireContains(t, out, "1 job", "the job count after a restart")
}

// TestCronStatusIgnoresAStalePidfile covers the second defect, which is the
// expensive one: it made the first invisible.
//
// `service cron status` reaches `pidofproc` in /lib/lsb/init-functions, and with
// a pidfile that function does no more than `kill -0` on the number it holds —
// it never checks the process is cron. /var/run/crond.pid is in the container's
// filesystem and survives a restart, so after one it names a pid from the
// previous boot; on a busy container that number belongs to something else, and
// cron is reported as running with no cron anywhere. Measured on a live project:
// `ps` had no cron, `madock status` said "Cron is running (6 jobs)", and the six
// jobs were real.
//
// The pidfile is planted through docker because nothing in madock can produce
// this state on purpose — and being able to produce it on purpose is the only
// way this stays fixed.
func TestCronStatusIgnoresAStalePidfile(t *testing.T) {
	p := newProject(t, "e2ecronpidfile")

	p.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2ecronpidfile.test",
	)
	p.run(20*time.Minute, "start")

	// pid 1 is the container's own init: alive for as long as the container is,
	// and never cron. That is exactly the shape of a recycled pid.
	inContainer(t, p, "app", "echo 1 > /var/run/crond.pid")

	if running, _ := cronFromJSON(t, p); running {
		t.Error("a pidfile naming pid 1 was read as a running cron daemon")
	}

	out, err := p.tryRun(3*time.Minute, "cron:status", "--json")
	if err != nil {
		t.Errorf("cron:status failed on a project with cron disabled: %v\n%s", err, out)
	}
	if strings.Contains(jsonPart(out), `"running": true`) {
		t.Errorf("cron:status believed the stale pidfile:\n%s", out)
	}
}

// installCronJob puts one job in the project's configuration.
//
// Written into the file rather than set with `config:set`: the defaults ship
// `cron/jobs` commented out, so the key does not exist and `config:set` refuses
// it by name. Editing config.xml by hand is how a project gets a cron job, which
// makes it the right thing for a test to do as well.
func installCronJob(t *testing.T, p *project, line string) {
	t.Helper()

	path := filepath.Join(p.execDir, "projects", p.name, "config.xml")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading the registry config: %v", err)
	}

	const anchor = "<cron>"
	if !strings.Contains(string(body), anchor) {
		t.Fatalf("no %s block in %s — the config layout changed", anchor, path)
	}

	updated := strings.Replace(string(body), anchor,
		anchor+"\n                <jobs>\n                    <job>"+line+"</job>\n                </jobs>", 1)
	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

// cronProcesses lists the cron processes a container actually has, read from
// /proc so the answer does not depend on `ps` being installed.
func cronProcesses(t *testing.T, p *project, service string) string {
	t.Helper()

	out := inContainer(t, p, service,
		`for c in /proc/[0-9]*/comm; do read n < "$c" 2>/dev/null || continue; case "$n" in cron|crond) echo "$n";; esac; done`)
	return strings.TrimSpace(out)
}

// inContainer runs a shell command in one of the project's containers as root.
//
// Through docker rather than through madock, and only ever for setting up or
// measuring a state: these tests drive madock and observe the container, and
// mixing the two would let a broken madock report itself healthy.
func inContainer(t *testing.T, p *project, service, script string) string {
	t.Helper()

	container := "madock_" + p.name + "-" + service + "-1"
	out, err := exec.Command("docker", "exec", "-u", "root", container, "sh", "-c", script).CombinedOutput()
	if err != nil {
		t.Fatalf("could not run %q in %s: %v\n%s", script, container, err, out)
	}
	return string(out)
}

// cronFromJSON reads both answers status gives about cron: what the
// configuration asks for, and what the container has.
func cronFromJSON(t *testing.T, p *project) (running, enabled bool) {
	t.Helper()

	var payload struct {
		Data struct {
			Tools struct {
				CronEnabled bool `json:"cron_enabled"`
				CronRunning bool `json:"cron_running"`
			} `json:"tools"`
		} `json:"data"`
	}

	out := p.run(3*time.Minute, "status", "--json")
	if err := json.Unmarshal([]byte(jsonPart(out)), &payload); err != nil {
		t.Fatalf("status --json did not decode: %v\n%s", err, out)
	}
	return payload.Data.Tools.CronRunning, payload.Data.Tools.CronEnabled
}
