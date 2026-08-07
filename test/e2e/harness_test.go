//go:build e2e

// Package e2e drives the real madock binary against a real Docker daemon.
//
// These are not unit tests and they are not fast. They exist to answer the one
// question the rest of the suite cannot: does the thing work at all. The golden
// tests prove that a config renders into the files we expect; only running it
// proves those files start a container, that the container is reachable, and
// that the command reports honestly about it.
//
// They never run on a developer's machine. See lima.yaml for why — in short,
// the proxy stack is named `aruntime` in the template rather than derived from
// anything, so a test would not conflict with your work, it would operate on
// the same containers.
//
//	./test/e2e/e2e.sh run
package e2e

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// project is one madock installation with one project in it, isolated from
// every other test by MADOCK_EXEC_DIR and MADOCK_RUN_DIR.
//
// Isolating the exec dir is what makes the tests independent: it holds the
// project registry, the generated runtime and the unpacked docker templates.
// Container names are not isolated by it, which is why each test picks its own
// project name — two tests sharing a name would share containers.
type project struct {
	t    *testing.T
	name string
	// execDir is the madock installation: registry, runtime, templates.
	execDir string
	// runDir is the project source directory. madock takes the project name
	// from its last path segment.
	runDir string
}

// installation is a madock install directory that projects can share. Most
// tests want one project and do not care; the proxy is the exception, because
// it is shared by every project in an installation and that sharing is the
// thing being tested.
type installation struct {
	t    *testing.T
	root string
	dir  string
}

func newInstallation(t *testing.T) *installation {
	t.Helper()

	binary()

	root := t.TempDir()
	i := &installation{t: t, root: root, dir: filepath.Join(root, "install")}
	if err := os.MkdirAll(i.dir, 0o755); err != nil {
		t.Fatalf("creating %s: %v", i.dir, err)
	}
	return i
}

// newProject prepares an empty project directory. Nothing is created inside
// madock yet — that is what `setup` does, and it is usually the first thing a
// test wants to assert about.
func newProject(t *testing.T, name string) *project {
	t.Helper()
	return newInstallation(t).project(name)
}

func (i *installation) project(name string) *project {
	t := i.t
	t.Helper()

	p := &project{
		t:       t,
		name:    name,
		execDir: i.dir,
		runDir:  filepath.Join(i.root, name),
	}
	if err := os.MkdirAll(p.runDir, 0o755); err != nil {
		t.Fatalf("creating %s: %v", p.runDir, err)
	}

	// Registered before setup runs, so a test that fails halfway still takes
	// its containers down. Without this a failing run leaves the daemon dirty
	// and the next run fails for a reason that has nothing to do with the code.
	t.Cleanup(p.destroy)

	return p
}

// run executes a madock command and fails the test if it does not succeed.
func (p *project) run(timeout time.Duration, args ...string) string {
	p.t.Helper()

	out, err := p.tryRun(timeout, args...)
	if err != nil {
		p.t.Fatalf("madock %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
	return out
}

// tryRun executes a madock command and hands back whatever happened. Use it
// when the failure is the thing being tested.
func (p *project) tryRun(timeout time.Duration, args ...string) (string, error) {
	p.t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, binary(), args...)
	cmd.Dir = p.runDir
	cmd.Env = append(os.Environ(),
		"MADOCK_EXEC_DIR="+p.execDir,
		"MADOCK_RUN_DIR="+p.runDir,
	)

	started := time.Now()
	out, err := cmd.CombinedOutput()
	p.t.Logf("madock %s (%s)", strings.Join(args, " "), time.Since(started).Round(time.Millisecond))

	if ctx.Err() == context.DeadlineExceeded {
		return string(out), context.DeadlineExceeded
	}
	return string(out), err
}

// destroy removes the project and everything Docker holds for it. Failures are
// reported rather than fatal: cleanup running after an already-failed test
// should not replace the real error with its own.
func (p *project) destroy() {
	if !p.configured() {
		return
	}
	if out, err := p.tryRun(5*time.Minute, "project:remove", "--force", "--name="+p.name); err != nil {
		p.t.Logf("cleanup of %s did not complete: %v\n%s", p.name, err, out)
	}
}

// configured reports whether setup ever got far enough to register the project.
func (p *project) configured() bool {
	_, err := os.Stat(filepath.Join(p.execDir, "projects", p.name, "config.xml"))
	return err == nil
}

// generated returns the path of a file in the project's generated runtime.
func (p *project) generated(relative string) string {
	return filepath.Join(p.execDir, "aruntime", "projects", p.name, relative)
}

// binary returns the madock under test, built for linux by e2e.sh.
func binary() string {
	path := os.Getenv("MADOCK_E2E_BIN")
	if path == "" {
		panic("MADOCK_E2E_BIN is not set — run these through ./test/e2e/e2e.sh run")
	}
	return path
}

func TestMain(m *testing.M) {
	if os.Getenv("MADOCK_E2E_BIN") == "" {
		// A plain `go test -tags=e2e ./...` on a laptop would otherwise start
		// containers on the developer's own daemon, which is exactly what this
		// package exists to avoid.
		println("MADOCK_E2E_BIN is not set — these tests run inside the VM: ./test/e2e/e2e.sh run")
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func requireContains(t *testing.T, haystack, needle, what string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("%s: expected to find %q in:\n%s", what, needle, haystack)
	}
}

func requireFile(t *testing.T, path, what string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Errorf("%s: %v", what, err)
	}
}
