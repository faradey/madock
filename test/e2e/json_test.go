//go:build e2e

package e2e

import (
	"encoding/json"
	"testing"
	"time"
)

// TestJSONOutputIsUsable checks the machine-readable half of madock.
//
// It is worth its own test because a broken `--json` looks like nothing. The
// command succeeds, prints its usual human output, exits zero — and the script
// reading it gets an empty string. That is not hypothetical: `db:export --json`
// printed no JSON at all for as long as the flag existed, because the argument
// struct declared a `Json` field while also embedding one, so the parser filled
// the copy the code was not reading.
//
// Which is the lesson this test encodes: assert that the payload arrived, not
// that the command was happy.
func TestJSONOutputIsUsable(t *testing.T) {
	p := newProject(t, "e2ejson")

	p.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2ejson.test",
	)
	p.run(20*time.Minute, "start")

	t.Run("status", func(t *testing.T) {
		var payload struct {
			Success bool `json:"success"`
			Data    struct {
				Services []struct {
					Service string `json:"service"`
					State   string `json:"state"`
					Running bool   `json:"running"`
				} `json:"services"`
				Tools struct {
					CronEnabled     bool `json:"cron_enabled"`
					DebuggerEnabled bool `json:"debugger_enabled"`
				} `json:"tools"`
			} `json:"data"`
		}

		out := p.run(3*time.Minute, "status", "--json")
		decode(t, out, &payload, "status --json")

		if len(payload.Data.Services) == 0 {
			t.Fatalf("status --json listed no services on a running project:\n%s", out)
		}

		// `running` is the field a script branches on. A JSON status that says
		// nothing is running while the project is up is worse than no JSON,
		// because something will act on it.
		anyRunning := false
		for _, service := range payload.Data.Services {
			if service.Running {
				anyRunning = true
			}
			if service.Service == "" {
				t.Errorf("a service came back without a name:\n%s", out)
			}
		}
		if !anyRunning {
			t.Errorf("status --json reports nothing running, but the project was just started:\n%s", out)
		}
	})

	t.Run("config:list", func(t *testing.T) {
		var payload struct {
			Success bool `json:"success"`
			Data    struct {
				Project string            `json:"project"`
				Config  map[string]string `json:"config"`
			} `json:"data"`
		}

		out := p.run(2*time.Minute, "config:list", "--json")
		decode(t, out, &payload, "config:list --json")

		if payload.Data.Project != "e2ejson" {
			t.Errorf("config:list --json names the project %q:\n%s", payload.Data.Project, out)
		}
		if len(payload.Data.Config) == 0 {
			t.Errorf("config:list --json returned an empty configuration:\n%s", out)
		}
		if payload.Data.Config["platform"] != "custom" {
			t.Errorf("config:list --json reports platform %q, not the one set up:\n%s",
				payload.Data.Config["platform"], out)
		}
	})
}

// decode unwraps madock's {success, data} envelope and fails helpfully when the
// output is not JSON at all — which is the failure this whole test exists for,
// so it should say so rather than complain about a character.
func decode(t *testing.T, out string, into any, what string) {
	t.Helper()

	payload := jsonPart(out)
	if payload == out {
		t.Fatalf("%s printed no JSON object:\n%s", what, out)
	}
	if err := json.Unmarshal([]byte(payload), into); err != nil {
		t.Fatalf("%s printed JSON that does not fit the expected shape: %v\n%s", what, err, out)
	}

	var envelope struct {
		Success bool `json:"success"`
	}
	if err := json.Unmarshal([]byte(payload), &envelope); err == nil && !envelope.Success {
		t.Fatalf("%s reported failure:\n%s", what, out)
	}
}
