package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

// Progress printed after Begin must not reach the stream the document goes
// to: a command that copies volumes prints as it goes, and a document mixed
// into that cannot be parsed.
func TestProgressGoesToStderrAndTheDocumentToStdout(t *testing.T) {
	realStdout, realStderr := os.Stdout, os.Stderr
	defer func() { os.Stdout, os.Stderr = realStdout, realStderr }()

	outR, outW, _ := os.Pipe()
	errR, errW, _ := os.Pipe()
	os.Stdout, os.Stderr = outW, errW

	w := Begin()
	fmt.Println("Stopping containers for a consistent copy...")
	w.Print(map[string]string{"name": "snapshot-2026"})

	outW.Close()
	errW.Close()
	var stdout, stderr bytes.Buffer
	stdout.ReadFrom(outR)
	stderr.ReadFrom(errR)

	var doc Response
	if err := json.Unmarshal(stdout.Bytes(), &doc); err != nil {
		t.Fatalf("stdout is not one JSON document: %v\n%s", err, stdout.String())
	}
	if !doc.Success {
		t.Errorf("unexpected document: %s", stdout.String())
	}
	if !bytes.Contains(stderr.Bytes(), []byte("Stopping containers")) {
		t.Errorf("the progress line should be on stderr, got %q", stderr.String())
	}
}

func TestFailWritesAFailureAndExitsOne(t *testing.T) {
	code := -1
	exit = func(c int) { code = c }
	defer func() { exit = os.Exit }()

	var buf bytes.Buffer
	(&Writer{out: &buf}).FailWith([]string{"db"}, "restore failed: %s\n", "disk full")

	var doc Response
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, buf.String())
	}
	if doc.Success || doc.Error != "restore failed: disk full" || doc.Data == nil {
		t.Errorf("unexpected document: %s", buf.String())
	}
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
}

// Rows that are JSON already are wrapped as they are, in the same document
// every other answer is: db:execute answered the bare array.
func TestRawDataIsWrappedInTheResponse(t *testing.T) {
	var out bytes.Buffer
	w := &Writer{out: &out}
	if err := w.PrintRaw(bytes.NewBufferString(`[{"v":"one"}]`)); err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Success bool                `json:"success"`
		Data    []map[string]string `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
		t.Fatalf("not one document: %v\n%s", err, out.String())
	}
	if !doc.Success || len(doc.Data) != 1 || doc.Data[0]["v"] != "one" {
		t.Errorf("got %s", out.String())
	}
}
