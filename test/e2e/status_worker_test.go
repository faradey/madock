//go:build e2e

package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestStatusListsAWorkerThatStopped — a worker that refused to start used to be
// missing from `madock status` altogether, because `compose ps` lists running
// containers only: the role looked absent, not failed (Indemorio IND-11). A
// stopped worker-<name> is now listed with its state and exit code, in the text
// and in --json; other stopped services stay out as before.
func TestStatusListsAWorkerThatStopped(t *testing.T) {
	p := newProject(t, "e2eworkerdown")
	p.run(5*time.Minute, "setup", "-y", "--platform=custom", "--language=none", "--hosts=e2eworkerdown.test")
	// Program names are the project's own keys, so they go in its
	// .madock/config.xml rather than through config:set.
	dir := filepath.Join(p.runDir, ".madock")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	config := `<?xml version="1.0" encoding="UTF-8"?>
<config>
    <scopes>
        <default>
            <worker>
                <enabled>true</enabled>
                <programs>
                    <broken>cat /nonexistent-e2e</broken>
                </programs>
            </worker>
        </default>
    </scopes>
</config>
`
	if err := os.WriteFile(filepath.Join(dir, "config.xml"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	p.run(20*time.Minute, "start")

	// The command fails at once (exit 1) — not `false`, which the template
	// reads as an empty program and leaves out; restart may bring it back for a moment, so the
	// state is read until it is a stopped one.
	var text string
	for i := 0; i < 20; i++ {
		text = p.run(time.Minute, "status")
		if strings.Contains(text, "worker-broken") && !strings.Contains(text, "worker-broken running") {
			break
		}
		time.Sleep(3 * time.Second)
	}
	requireContains(t, text, "worker-broken", "a worker that exited, in the text")
	requireContains(t, text, "(exit 1)", "its exit code")

	raw := p.run(time.Minute, "status", "--json")
	var out struct {
		Data struct {
			Services []struct {
				Service  string `json:"service"`
				Running  bool   `json:"running"`
				ExitCode *int   `json:"exit_code"`
			} `json:"services"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(raw[strings.Index(raw, "{"):]), &out); err != nil {
		t.Fatalf("status --json: %v\n%s", err, raw)
	}
	found := false
	for _, s := range out.Data.Services {
		if s.Service == "worker-broken" {
			found = true
			if s.Running || s.ExitCode == nil || *s.ExitCode != 1 {
				t.Errorf("worker-broken in --json: running=%v exit_code=%v", s.Running, s.ExitCode)
			}
		}
	}
	if !found {
		t.Errorf("worker-broken missing from --json:\n%s", raw)
	}
}
