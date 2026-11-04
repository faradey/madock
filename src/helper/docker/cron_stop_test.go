package docker

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runCronStopScript runs cronStopScript under /bin/sh with PATH limited to dir,
// the way it runs in a container that has only what dir holds.
func runCronStopScript(t *testing.T, dir string) (string, error) {
	t.Helper()
	cmd := exec.Command("/bin/sh", "-c", cronStopScript)
	cmd.Env = []string{"PATH=" + dir}
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// An image with no `service` — Packeton's Alpine one — has no daemon madock
// could have started, and every rebuild with cron disabled printed the failed
// exec. The stop must succeed there and say nothing.
func TestCronStopIsQuietWithoutService(t *testing.T) {
	out, err := runCronStopScript(t, t.TempDir())
	if err != nil {
		t.Fatalf("stopping cron failed in an image without `service`: %v\n%s", err, out)
	}
	if out != "" {
		t.Errorf("stopping cron printed in an image without `service`:\n%s", out)
	}
}

// Where `service` exists the daemon is still stopped: the guard must not turn
// the stop into nothing everywhere.
func TestCronStopCallsServiceWhereItExists(t *testing.T) {
	dir := t.TempDir()
	called := filepath.Join(dir, "called")
	fake := "#!/bin/sh\necho \"$@\" > " + called + "\n"
	if err := os.WriteFile(filepath.Join(dir, "service"), []byte(fake), 0o755); err != nil {
		t.Fatal(err)
	}

	if out, err := runCronStopScript(t, dir); err != nil {
		t.Fatalf("stopping cron failed: %v\n%s", err, out)
	}

	got, err := os.ReadFile(called)
	if err != nil {
		t.Fatalf("`service` was not called: %v", err)
	}
	if strings.TrimSpace(string(got)) != "cron stop" {
		t.Errorf("`service` was called with %q, want \"cron stop\"", strings.TrimSpace(string(got)))
	}
}
