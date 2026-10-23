//go:build e2e

package e2e

import (
	"encoding/json"
	"testing"
	"time"
)

// TestDbExecuteJSONIsOneDocument — db:execute --json answered the bare array
// of rows, the one --json command outside {success, data}, so a caller reading
// success got nothing and one reading data got nothing either (found by the
// Indemorio engine, 2026-10-23). And a query that fails must answer
// {"success": false}, not half an array.
func TestDbExecuteJSONIsOneDocument(t *testing.T) {
	p := newProject(t, "e2edbjson")
	p.run(setupTimeout, "setup", "-y", "--platform=custom", "--language=none", "--hosts=e2edbjson.test")
	p.run(20*time.Minute, "start")
	p.freshTable("probe", "(v VARCHAR(32))")
	p.query("INSERT INTO probe VALUES ('one'), ('two')")

	out := p.run(2*time.Minute, "db:execute", "--json", "SELECT v FROM probe ORDER BY v")
	var answer struct {
		Success bool                `json:"success"`
		Data    []map[string]string `json:"data"`
	}
	if err := json.Unmarshal([]byte(jsonPart(out)), &answer); err != nil {
		t.Fatalf("db:execute --json is not one document: %v\n%s", err, out)
	}
	if !answer.Success || len(answer.Data) != 2 || answer.Data[0]["v"] != "one" || answer.Data[1]["v"] != "two" {
		t.Errorf("db:execute --json answered %+v, want success and the two rows", answer)
	}

	out, err := p.tryRun(2*time.Minute, "db:execute", "--json", "SELECT nope FROM no_such_table")
	if err == nil {
		t.Fatalf("a failing query exited 0:\n%s", out)
	}
	var failed struct {
		Success *bool  `json:"success"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal([]byte(jsonPart(out)), &failed); err != nil || failed.Success == nil || *failed.Success || failed.Error == "" {
		t.Errorf("a failing query did not answer {success: false, error}: %v\n%s", err, out)
	}
}
