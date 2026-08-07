//go:build e2e

package e2e

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

// TestDumpRoundTrip exports the database to a file and imports it back.
//
// The dump is the thing people hand to each other and to us: it travels between
// a server and a laptop, and it is what a support conversation starts with. So
// two failures matter here and they are different. An export that produces
// nothing while exiting zero is the quieter one — the file is noticed to be
// empty weeks later, by which time the database it came from has moved on.
func TestDumpRoundTrip(t *testing.T) {
	p := newProject(t, "e2edump")

	p.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2edump.test",
	)
	p.run(20*time.Minute, "start")

	p.freshTable("probe", "(note VARCHAR(32))")
	p.query("INSERT INTO probe VALUES ('before-dump')")

	// --json exists so a script does not have to guess the name: the file
	// carries a timestamp the caller never chose.
	out := p.run(10*time.Minute, "db:export", "-n", "probe", "--json")

	// Every JSON answer madock gives is wrapped in {success, data}; the payload
	// is inside. A test that reads the payload directly would pass on a command
	// that reported failure.
	var exported struct {
		Success bool `json:"success"`
		Data    struct {
			File string `json:"file"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(jsonPart(out)), &exported); err != nil {
		t.Fatalf("db:export --json did not print usable JSON: %v\n%s", err, out)
	}
	if !exported.Success {
		t.Fatalf("db:export --json reported failure:\n%s", out)
	}
	if exported.Data.File == "" {
		t.Fatalf("db:export --json reported no file:\n%s", out)
	}

	info, err := os.Stat(exported.Data.File)
	if err != nil {
		t.Fatalf("the dump db:export reported does not exist: %v", err)
	}
	// A gzip stream of nothing is still about twenty bytes, so "not empty" is
	// not enough of a bar to be worth stating loosely.
	if info.Size() < 100 {
		t.Errorf("the dump is %d bytes, which is too small to contain a database", info.Size())
	}

	p.query("INSERT INTO probe VALUES ('after-dump')")

	p.run(10*time.Minute, "db:import", "--force", exported.Data.File)

	restored := p.query("SELECT note FROM probe")
	requireContains(t, restored, "before-dump", "the row the dump contained")
	if strings.Contains(restored, "after-dump") {
		t.Errorf("the row written after the dump survived the import:\n%s", restored)
	}
}

// jsonPart returns the JSON object in a command's output.
//
// --json is a flag on a human-facing command, not a separate mode: progress
// lines can be printed before the payload, and a parser that assumes otherwise
// fails on a machine that happened to pull an image.
func jsonPart(out string) string {
	start := strings.Index(out, "{")
	end := strings.LastIndex(out, "}")
	if start < 0 || end < start {
		return out
	}
	return out[start : end+1]
}
