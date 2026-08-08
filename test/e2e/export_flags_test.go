//go:build e2e

package e2e

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

// TestExportIgnoreTableLeavesTheTableOut checks a flag by reading what it
// produced, which for an export is the only honest way.
//
// --ignore-table is the shape of flag that fails quietly: the command succeeds,
// the dump is written, and the table is in it anyway. Nobody looks inside a
// gzip until they restore it somewhere they should not have — a dump handed to
// support with the table somebody meant to leave out, or a "cleaned" dump that
// still carries the customer data it was cleaned of.
func TestExportIgnoreTableLeavesTheTableOut(t *testing.T) {
	p := newProject(t, "e2eignore")

	p.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2eignore.test",
	)
	p.run(20*time.Minute, "start")

	p.freshTable("keepme", "(note VARCHAR(32))")
	p.freshTable("skipme", "(secret VARCHAR(32))")
	p.query("INSERT INTO keepme VALUES ('in-the-dump')")
	p.query("INSERT INTO skipme VALUES ('not-in-the-dump')")

	full := dumpContents(t, p, p.run(10*time.Minute, "db:export", "-n", "full", "--json"))
	requireContains(t, full, "keepme", "a plain dump should contain every table")
	requireContains(t, full, "skipme", "a plain dump should contain every table")

	partial := dumpContents(t, p, p.run(10*time.Minute, "db:export", "-n", "partial", "--ignore-table", "skipme", "--json"))
	requireContains(t, partial, "keepme", "the table that was not ignored")

	// Both the schema and the row. mysqldump can omit CREATE TABLE and still
	// carry the data, which would be the worst half to miss.
	if strings.Contains(partial, "CREATE TABLE `skipme`") {
		t.Error("the ignored table's schema is in the dump")
	}
	if strings.Contains(partial, "not-in-the-dump") {
		t.Error("the ignored table's rows are in the dump")
	}
}

// dumpContents reads the gzip the export just reported and returns it as text.
func dumpContents(t *testing.T, p *project, exportOutput string) string {
	t.Helper()

	var payload struct {
		Data struct {
			File string `json:"file"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(jsonPart(exportOutput)), &payload); err != nil {
		t.Fatalf("db:export --json did not decode: %v\n%s", err, exportOutput)
	}
	if payload.Data.File == "" {
		t.Fatalf("db:export --json reported no file:\n%s", exportOutput)
	}

	file, err := os.Open(payload.Data.File)
	if err != nil {
		t.Fatalf("opening the dump: %v", err)
	}
	defer file.Close()

	reader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatalf("the dump is not gzip: %v", err)
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("reading the dump: %v", err)
	}
	return string(content)
}
