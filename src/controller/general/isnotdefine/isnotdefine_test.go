package isnotdefine

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// Execute ends the process, so it runs in a copy of this test binary.
func TestAnUnknownCommandFailsOnStderr(t *testing.T) {
	if os.Getenv("MADOCK_TEST_UNKNOWN_COMMAND") == "1" {
		Execute("no-such-command")
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestAnUnknownCommandFailsOnStderr$")
	cmd.Dir = t.TempDir()
	cmd.Env = append(os.Environ(), "MADOCK_TEST_UNKNOWN_COMMAND=1")
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr

	err := cmd.Run()

	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 1 {
		t.Errorf("exit: %v, want status 1", err)
	}
	if !strings.Contains(stderr.String(), "not defined") {
		t.Errorf("stderr: %q, want the refusal", stderr.String())
	}
	if strings.Contains(stdout.String(), "not defined") {
		t.Errorf("stdout holds the refusal: %q", stdout.String())
	}
}
