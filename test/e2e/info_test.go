//go:build e2e

package e2e

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// dbPassword is deliberately unlike anything madock generates, so that finding
// it in command output means one thing only.
//
// It is set with `config:set` after setup, which is safe here because this test
// never connects to the database: the container initialises its data directory
// from the credentials in the generated compose file, so a value written
// afterwards is visible to every command and to nothing else. A test that
// queries has to read the generated password instead — see p.configValue.
const dbPassword = "e2e-plaintext-password"

// TestInfoReportsWhatIsActuallyPublished treats `info` and `info:ports` as the
// contract they are.
//
// Neither has a test, and both are read by things that are not people: scripts
// connect to the database on the port `info:ports` prints, and `info` is the
// first thing anyone runs when something does not add up. Two ways of being
// wrong matter here, and only one of them is loud.
//
//   - printing a port nothing is listening on. Whoever reads it concludes the
//     database is down.
//   - allocating a port while answering. The source says so in as many words —
//     "Read-only lookup — do not allocate a port from `madock info`" — because a
//     command people run to *look* must not change what it is looking at. A
//     project that has never started has no ports, and asking about it must not
//     give it one.
func TestInfoReportsWhatIsActuallyPublished(t *testing.T) {
	p := newProject(t, "e2einfo")

	p.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2einfo.test",
	)

	// Recognisable on sight, so that a leak into the printed output cannot be
	// mistaken for anything else.
	p.run(2*time.Minute, "config:set", "-n", "db/password", "-v", dbPassword)

	// Before the first start. Whatever the registry says now, both commands
	// have to leave it saying exactly that.
	registry := filepath.Join(p.execDir, "aruntime", "ports.conf")
	before := readIfExists(t, registry)

	p.run(2*time.Minute, "info")
	p.run(2*time.Minute, "info:ports")

	if after := readIfExists(t, registry); after != before {
		t.Errorf("asking for information allocated a port.\nbefore:\n%s\nafter:\n%s", before, after)
	}

	p.run(20*time.Minute, "start")

	// --json is what a script reads, so it is the half worth pinning.
	var reported struct {
		Success bool `json:"success"`
		Data    struct {
			Project string `json:"project"`
			Ports   []struct {
				Service string `json:"service"`
				Port    int    `json:"port"`
			} `json:"ports"`
		} `json:"data"`
	}
	out := p.run(2*time.Minute, "info:ports", "--json")
	if err := json.Unmarshal([]byte(jsonPart(out)), &reported); err != nil {
		t.Fatalf("info:ports --json did not decode: %v\n%s", err, out)
	}
	if !reported.Success {
		t.Fatalf("info:ports --json reported failure:\n%s", out)
	}
	if reported.Data.Project != p.name {
		t.Errorf("info:ports --json names project %q, not %q", reported.Data.Project, p.name)
	}

	dbPort := 0
	for _, entry := range reported.Data.Ports {
		if entry.Service == "db" {
			dbPort = entry.Port
		}
	}
	if dbPort == 0 {
		t.Fatalf("info:ports --json listed no database port:\n%s", out)
	}

	// The assertion that separates a number from a fact: something answers
	// there. A registry entry left behind by an earlier project would still
	// print, and would still be wrong.
	conn, err := net.DialTimeout("tcp", "127.0.0.1:"+strconv.Itoa(dbPort), 15*time.Second)
	if err != nil {
		t.Errorf("nothing is listening on the database port info:ports reported (%d): %v", dbPort, err)
	} else {
		_ = conn.Close()
	}

	// `info` prints the same port as the database's remote address, and the two
	// are produced by different code. They have to agree, or one of the two
	// commands is telling somebody the wrong thing.
	human := p.run(2*time.Minute, "info")
	requireContains(t, human, p.name, "info naming the project")
	requireContains(t, human, "e2einfo.test", "info naming the host")
	requireContains(t, human, "localhost:"+strconv.Itoa(dbPort), "info agreeing with info:ports about the database port")

	// The password is printed in full, and in this edition that is the intended
	// answer: madock manages the machine the person is sitting at, and `info` is
	// how they read their own project's credentials back. The value was set to
	// something recognisable at setup time, so a masked one would be
	// unmistakable here too — which is what the paid edition's own suite asserts
	// instead, for the servers it runs on.
	if !strings.Contains(human, dbPassword) {
		t.Errorf("info did not print the database password this edition is supposed to show:\n%s", human)
	}
	requireContains(t, human, "password", "info showing that there is a password at all")
}

// readIfExists returns a file's contents, or "" when it is not there. The ports
// registry does not exist until something allocates, and "not there" is a
// perfectly good before-state.
func readIfExists(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ""
		}
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(data)
}
