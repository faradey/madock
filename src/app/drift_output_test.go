package app

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/faradey/madock/v4/src/helper/embedded"
)

// The warning belongs on stderr and the reason is not tidiness: on a source
// checkout drift is the ordinary state of a session spent editing templates, and
// five lines in front of `project:list --json` or `db:export --json` are not a
// warning any more — they are malformed output, and every consumer stops parsing
// at the first character.
func TestDriftWarningStaysOffStdout(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MADOCK_EXEC_DIR", dir)

	// The embed declaration is what says this tree is a clone rather than an
	// extraction, and only a clone can drift.
	mustWrite(t, filepath.Join(dir, "docker", "embed.go"), "package dockerassets\n")
	mustWrite(t, filepath.Join(dir, "docker", "general", "docker-compose.yml"), "edited on disk\n")

	embedded.DockerFS = fstest.MapFS{
		"general/docker-compose.yml": {Data: []byte("as the binary was built\n")},
	}
	t.Cleanup(func() { embedded.DockerFS = nil })

	stdout, stderr := capture(t, reportTemplateDrift)

	if stdout != "" {
		t.Errorf("the warning went to stdout, so any --json output is now unparseable:\n%s", stdout)
	}
	if !strings.Contains(stderr, "different templates") {
		t.Errorf("the warning did not reach stderr either, so nothing was said at all:\n%s", stderr)
	}
	if !strings.Contains(stderr, "docker/general/docker-compose.yml") {
		t.Errorf("the differing template was not named:\n%s", stderr)
	}
}

// A binary built from the templates beside it says nothing at all — on either
// stream. A warning that is always there is one nobody reads.
func TestNoDriftSaysNothing(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MADOCK_EXEC_DIR", dir)

	mustWrite(t, filepath.Join(dir, "docker", "embed.go"), "package dockerassets\n")
	mustWrite(t, filepath.Join(dir, "docker", "general", "docker-compose.yml"), "as the binary was built\n")

	embedded.DockerFS = fstest.MapFS{
		"general/docker-compose.yml": {Data: []byte("as the binary was built\n")},
	}
	t.Cleanup(func() { embedded.DockerFS = nil })

	stdout, stderr := capture(t, reportTemplateDrift)

	if stdout != "" || stderr != "" {
		t.Errorf("a binary built from these very templates said something:\nstdout: %s\nstderr: %s", stdout, stderr)
	}
}

func TestWriteTemplateDriftCountsAndTrims(t *testing.T) {
	drift := &embedded.Drift{
		Changed: []string{"general/a.yml", "general/b.yml"},
		Missing: []string{"snippets/gone"},
		Extra:   []string{"snippets/new", "snippets/newer"},
	}

	var out bytes.Buffer
	writeTemplateDrift(&out, drift, "/opt/madock")
	body := out.String()

	if !strings.Contains(body, "2 changed, 1 missing from disk, 2 new to it") {
		t.Errorf("the counts are wrong:\n%s", body)
	}
	// Three named, the rest counted — the message has to fit on a screen.
	if !strings.Contains(body, "and 2 more") {
		t.Errorf("the remainder was not counted:\n%s", body)
	}
	if !strings.Contains(body, "go build -o madock . in /opt/madock") {
		t.Errorf("the fix does not name the directory to run it in:\n%s", body)
	}
}

func mustWrite(t *testing.T, path, body string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// capture runs f with both standard streams replaced, and returns what each of
// them received. Separately, because "which stream" is the whole question here.
func capture(t *testing.T, f func()) (stdout, stderr string) {
	t.Helper()

	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	errR, errW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	realOut, realErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = outW, errW

	// Read while f runs: a pipe holds about 64 KiB, and a test that fills it
	// would deadlock instead of failing.
	outDone := make(chan string, 1)
	errDone := make(chan string, 1)
	go func() { body, _ := io.ReadAll(outR); outDone <- string(body) }()
	go func() { body, _ := io.ReadAll(errR); errDone <- string(body) }()

	f()

	outW.Close()
	errW.Close()
	os.Stdout, os.Stderr = realOut, realErr

	return <-outDone, <-errDone
}
