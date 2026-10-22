//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestARefusalBeforeTheCommandLeavesStdoutEmpty runs a JSON command from a
// directory inside an already registered project — a Deployer `current`, in
// real life — where madock refuses before the command is reached.
//
// That refusal is printed by the configuration layer with fmt and ends in
// os.Exit, so it used to reach stdout as a page of coloured text where one
// JSON document was promised. The contract says a failure that the command
// could not answer leaves stdout empty and exits non-zero; this is the case
// that broke it, measured on a production machine on 2026-10-15.
func TestARefusalBeforeTheCommandLeavesStdoutEmpty(t *testing.T) {
	p := newProject(t, "jsonrefusal")
	p.run(setupTimeout, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=jsonrefusal.test",
	)

	inner := filepath.Join(p.runDir, "inner")
	if err := os.MkdirAll(filepath.Join(inner, ".madock"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(inner, ".madock", "config.xml"), []byte("<config>\n</config>\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary(), "status", "--json")
	cmd.Dir = inner
	cmd.Env = append(os.Environ(), "MADOCK_EXEC_DIR="+p.execDir, "MADOCK_RUN_DIR="+inner)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	var exit *exec.ExitError
	if !errors.As(err, &exit) {
		t.Fatalf("status --json from inside a registered project should fail, got %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
	if out := strings.TrimSpace(stdout.String()); out != "" {
		t.Errorf("stdout should be empty when madock refuses before the command, got:\n%s", out)
	}
	requireContains(t, stderr.String(), "already registered", "the refusal should still be said, on stderr")
}
