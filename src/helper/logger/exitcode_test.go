package logger

import (
	"errors"
	"os/exec"
	"strconv"
	"testing"
)

// The bug: every failure of a program madock ran came back as 1.
//
//	madock cli bash -c "exit 137"  → 1
//	madock cli bash -c "exit 3"    → 1
//
// 137 is the OOM killer and means "give it more memory"; 1 from a test runner
// means "fix the code". A script could tell that something failed and nothing
// about what. madock had the number the whole time — exec.Cmd.Run returns an
// *exec.ExitError carrying it, and the debug log even printed it.
//
// The exit itself cannot be tested in-process, so this covers the decision that
// precedes it: which code FatalChild would use.
func TestChildExitCode(t *testing.T) {
	cases := []struct {
		name string
		code int
		want int
	}{
		{"the OOM killer", 137, 137},
		{"an ordinary failure", 3, 3},
		{"one, which is also a real code", 1, 1},
	}

	for _, c := range cases {
		err := runExiting(t, c.code)
		if got := childExitCode(err); got != c.want {
			t.Errorf("%s: childExitCode = %d, want %d", c.name, got, c.want)
		}
	}
}

// Anything that is not a child's failure is madock failing, and that is 1.
// Inventing a code for it would make the number mean two things.
func TestChildExitCodeFallsBackToOne(t *testing.T) {
	if got := childExitCode(errors.New("docker is not running")); got != 1 {
		t.Errorf("a plain error gave %d, want 1", got)
	}
	if got := childExitCode(nil); got != 1 {
		t.Errorf("no error at all gave %d, want 1", got)
	}
}

// runExiting runs a child that exits with the given code and returns its error.
func runExiting(t *testing.T, code int) error {
	t.Helper()

	cmd := exec.Command("sh", "-c", "exit "+strconv.Itoa(code))
	err := cmd.Run()
	if err == nil {
		t.Fatalf("a child exiting %d returned no error", code)
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected an *exec.ExitError, got %T", err)
	}
	return err
}
