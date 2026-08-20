//go:build e2e

package e2e

import (
	"strings"
	"testing"
	"time"
)

// TestStatusNamesAContainerTheConfigurationDropped is the lie this was written
// against, reproduced the way it happened.
//
// `docker compose ps` lists by project, not by file: the project name comes from
// the directory the compose file sits in, so a container created from an earlier
// version of that file keeps being reported after the service is gone from it.
// `status` presented those as ordinary services.
//
// It hid a real defect for a day. A project whose config said `db/type: MariaDB`
// generated a compose file with **no db service at all**, and `status --json`
// went on listing `db` as running — so the first version of the test written for
// that bug passed against the broken build, and only reading the generated file
// showed the truth. A status that invents a service is worse than one that says
// nothing, because it is believed.
//
// Nothing is removed here, and that is the decision rather than an omission. The
// compose file is generated from a config, so a rendering bug decides what counts
// as an orphan — `--remove-orphans` on `up` would have pointed the deletion at a
// running database container in exactly the case above.
func TestStatusNamesAContainerTheConfigurationDropped(t *testing.T) {
	p := newProject(t, "e2eorphan")

	p.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2eorphan.test",
	)
	p.run(20*time.Minute, "start")

	if !p.runningServices(t, "after start")["db"] {
		t.Fatal("the project came up without a database, so this test has nothing to orphan")
	}

	// The service leaves the configuration. The container does not leave with it
	// — that is the state under test, and it is an ordinary one: `db/enabled` is
	// how a project says it has no database of its own, and any project moving to
	// a shared database goes through exactly this.
	p.run(2*time.Minute, "config:set", "-n", "db/enabled", "-v", "false")
	p.run(20*time.Minute, "start")

	generated := readFile(t, p.generated("docker-compose.yml"))
	if strings.Contains(generated, "\n  db:") {
		t.Fatalf("db is still declared in the compose file, so nothing was orphaned:\n%s", generated)
	}

	// The text output has to say the word, because the failure is that a reader
	// takes the line for a working service.
	human := p.run(2*time.Minute, "status")
	if strings.Contains(human, "db running") && !strings.Contains(human, "orphan") {
		t.Errorf("status listed db as a service of this project without naming it a leftover:\n%s", human)
	}

	// And the JSON, which is what a script reads. `orphan` is absent for an
	// ordinary service and present here, so its presence is the exception.
	var payload struct {
		Data struct {
			Services []struct {
				Service string `json:"service"`
				Running bool   `json:"running"`
				Orphan  bool   `json:"orphan"`
			} `json:"services"`
		} `json:"data"`
	}
	decode(t, p.run(3*time.Minute, "status", "--json"), &payload, "status --json")

	for _, service := range payload.Data.Services {
		if service.Service == "db" && !service.Orphan {
			t.Errorf("status --json reported db as a service of this project: %+v", payload.Data.Services)
		}
		if service.Service == "app" && service.Orphan {
			t.Errorf("app is declared and was flagged as a leftover: %+v", payload.Data.Services)
		}
	}
}

// runningServices reads the project's own services out of `status --json`.
func (p *project) runningServices(t *testing.T, what string) map[string]bool {
	t.Helper()

	var payload struct {
		Data struct {
			Services []struct {
				Service string `json:"service"`
				Running bool   `json:"running"`
			} `json:"services"`
		} `json:"data"`
	}
	decode(t, p.run(3*time.Minute, "status", "--json"), &payload, "status --json "+what)

	running := map[string]bool{}
	for _, service := range payload.Data.Services {
		running[service.Service] = service.Running
	}
	return running
}
