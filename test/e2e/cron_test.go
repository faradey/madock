//go:build e2e

package e2e

import (
	"encoding/json"
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
