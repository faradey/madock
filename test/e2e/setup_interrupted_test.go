//go:build e2e

package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestASetupKilledHalfwayIsFinishedByTheNextOne — `setup -d -i` killed in the
// middle, the way a dropped ssh session or a Ctrl-C ends it: the starter is
// cloned, the dependency install is running, and madock is gone. Running the
// same setup again has to finish the project rather than trip over what the
// first one left (a non-empty directory to clone into, a half-filled
// node_modules, a config that already exists), and the project has to answer.
//
// Medusa because it installs with no credentials and its install is long
// enough to be caught in the middle of.
func TestASetupKilledHalfwayIsFinishedByTheNextOne(t *testing.T) {
	if !platformTestsEnabled() {
		t.Skip("platform tests are opt-in: ./test/e2e/e2e.sh run --platforms -run TestASetupKilledHalfway")
	}
	p := newProject(t, "e2ehalfway")
	args := []string{"setup", "-y", "-d", "-i", "--platform=medusa", "--hosts=e2ehalfway.test"}

	cmd := exec.Command(binary(), args...)
	cmd.Dir = p.runDir
	cmd.Env = append(os.Environ(), "MADOCK_EXEC_DIR="+p.execDir, "MADOCK_RUN_DIR="+p.runDir)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	log, err := os.Create(filepath.Join(t.TempDir(), "first-setup.log"))
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stdout, cmd.Stderr = log, log
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	// Killed once the clone is there and the install has begun writing
	// node_modules: the middle, not the start or the end.
	deadline := time.Now().Add(30 * time.Minute)
	caught := false
	for time.Now().Before(deadline) {
		if _, err := os.Stat(filepath.Join(p.runDir, "node_modules")); err == nil {
			caught = true
			break
		}
		if cmd.ProcessState != nil {
			break
		}
		time.Sleep(2 * time.Second)
	}
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	_ = cmd.Wait()
	if !caught {
		body, _ := os.ReadFile(log.Name())
		t.Fatalf("the first setup never reached the dependency install, so nothing was interrupted:\n%s", firstLines(string(body), 60))
	}
	t.Logf("first setup killed with node_modules half written: %s", describeTree(t, filepath.Join(p.runDir, "node_modules")))

	// The same command again, as anybody would run it.
	p.run(45*time.Minute, args...)

	requireFile(t, filepath.Join(p.runDir, "package.json"), "the starter after the second setup")
	requireFile(t, filepath.Join(p.runDir, "storefront", "package.json"), "the storefront after the second setup")
	status, body := httpsGet(t, "e2ehalfway.test", "/health")
	if status != 200 || !strings.Contains(strings.ToUpper(body), "OK") {
		t.Fatalf("after a setup killed halfway and run again, /health answered %d:\n%s", status, firstLines(body, 20))
	}
}
