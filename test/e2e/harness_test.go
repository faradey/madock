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
	"strconv"
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
	t       *testing.T
	install *installation
	name    string
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
	t        *testing.T
	root     string
	dir      string
	projects []*project

	proxyReset     bool
	proxyDestroyed bool
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

// resetProxy removes whatever proxy is running before this installation starts
// anything, once.
//
// The teardown at the end of a test is not enough on its own: a run that is
// interrupted, or that fails before its cleanup, leaves the container up. The
// next run then inherits its certificate and routing and reports on state
// nobody created — a two-project test once passed against a binary with a known
// defect for exactly this reason, and later failed while being served a
// certificate belonging to a project from a different test entirely.
//
// Called before the first `start`, when setup has already written the project
// config the command needs.
func (i *installation) resetProxy(from *project) {
	if i.proxyReset {
		return
	}
	i.proxyReset = true

	if out, err := from.tryRun(3*time.Minute, "proxy:prune", "--force"); err != nil {
		i.t.Logf("could not remove a leftover proxy: %v\n%s", err, out)
	}
}

// destroyProxy removes the shared proxy container, once per installation.
//
// Everything else a test creates is isolated by MADOCK_EXEC_DIR, but the proxy
// is one container per Docker daemon and it outlives the installation that
// started it. Leaving it up lets a later test inherit an earlier one's
// certificate and routing — which is not a theory: a binary built before the
// certificate fix passed this suite once, on state a fixed binary had left
// behind a minute earlier. The test only started telling the two apart after
// this.
//
// It runs from a project that still exists, before any of them is removed:
// `project:remove` deletes the project directory, and madock has nowhere to be
// when its working directory is gone.
func (i *installation) destroyProxy(from *project) {
	if i.proxyDestroyed {
		return
	}
	i.proxyDestroyed = true

	if out, err := from.tryRun(3*time.Minute, "proxy:prune", "--force"); err != nil {
		i.t.Logf("could not remove the proxy: %v\n%s", err, out)
	}
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
		install: i,
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

	i.projects = append(i.projects, p)

	return p
}

// run executes a madock command and fails the test if it does not succeed.
func (p *project) run(timeout time.Duration, args ...string) string {
	p.t.Helper()

	if len(args) > 0 && args[0] == "start" {
		p.install.resetProxy(p)
	}

	out, err := p.tryRun(timeout, args...)
	if err != nil {
		p.t.Fatalf("madock %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
	return out
}

// runWithInput executes a madock command that asks a question, answering it
// from the string given. Commands that cannot be driven by flags — snapshot
// restore chooses from a numbered list and has none — can only be tested this
// way.
func (p *project) runWithInput(timeout time.Duration, input string, args ...string) string {
	p.t.Helper()

	out, err := p.tryRunWithInput(timeout, input, args...)
	if err != nil {
		p.t.Fatalf("madock %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
	return out
}

// tryRun executes a madock command and hands back whatever happened. Use it
// when the failure is the thing being tested.
func (p *project) tryRun(timeout time.Duration, args ...string) (string, error) {
	return p.tryRunWithInput(timeout, "", args...)
}

func (p *project) tryRunWithInput(timeout time.Duration, input string, args ...string) (string, error) {
	p.t.Helper()
	return p.tryRunWith(timeout, input, nil, args...)
}

// tryRunWith is the full form: input on stdin and extra environment variables.
//
// The environment is not decoration for these tests — MADOCK_USER,
// MADOCK_SERVICE_NAME and MADOCK_WORKDIR are a documented way to redirect any
// exec-shaped command, and they are read in one helper that every one of those
// commands calls. Nothing can drive them except from outside the process.
func (p *project) tryRunWith(timeout time.Duration, input string, env []string, args ...string) (string, error) {
	p.t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, binary(), args...)
	cmd.Dir = p.runDir
	if input != "" {
		cmd.Stdin = strings.NewReader(input)
	}
	cmd.Env = append(os.Environ(),
		"MADOCK_EXEC_DIR="+p.execDir,
		"MADOCK_RUN_DIR="+p.runDir,
	)
	cmd.Env = append(cmd.Env, env...)

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
		// A test that removed its own project — because removal was the thing
		// it was testing — lands here, and it used to mean the proxy was never
		// taken down. That container is one per daemon and outlives the
		// installation that started it, so the next test inherited a proxy
		// serving somebody else's configuration and its site answered nothing.
		// A whole CI failure was spent finding that out. A test that removes its
		// project must take the proxy with it first, and this says so out loud
		// rather than leaving the next one to discover it.
		if !p.install.proxyDestroyed {
			p.t.Logf("%s was removed by the test itself, so the shared proxy was never taken down — "+
				"call install.destroyProxy(p) before project:remove", p.name)
		}
		return
	}
	p.install.destroyProxy(p)
	if out, err := p.tryRun(5*time.Minute, "project:remove", "--force", "--name="+p.name); err != nil {
		p.t.Logf("cleanup of %s did not complete: %v\n%s", p.name, err, out)
	}

	// Containers write as root, and some of what they write outlives them: a
	// Medusa install leaves `.medusa/client/` owned by root inside the project.
	// Go's own TempDir cleanup then fails with "permission denied" and marks
	// the test failed after it has already passed — a red result whose cause is
	// housekeeping. Removing it as root is the only way, and it is confined to
	// a directory this test created.
	if out, err := exec.Command("sudo", "rm", "-rf", p.runDir).CombinedOutput(); err != nil {
		p.t.Logf("could not remove %s as root: %v\n%s", p.runDir, err, out)
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

// query runs SQL and waits for the database to be ready to answer.
//
// A container that exists is not a database that accepts connections, and the
// wait is different on the first run of the day from every run after it — so a
// fixed sleep is either flaky or wasteful.
// dbReadyTimeout is how long a database is given to start accepting connections.
//
// Five minutes rather than three because a runner starting two databases at once
// missed the old deadline: the first database's very first query failed the whole
// test after three minutes of `ERROR 2002 (HY000): Can't connect to server`. On a
// laptop the same query answers in seconds, which is why the limit had never been
// reached before CI ran the second-database tests.
const dbReadyTimeout = 5 * time.Minute

func (p *project) query(sql string) string {
	p.t.Helper()
	return p.queryOn("db", sql)
}

// queryOn runs a query against one of the project's databases, waiting for it to
// come up. A container that exists is not a server accepting connections, and every
// caller here has just started or restarted one.
//
// The failure says how long it waited and how many attempts it made, because "never
// succeeded" alone leaves the reader unable to tell a database that never came up
// from a query that was wrong all along.
func (p *project) queryOn(service, sql string) string {
	p.t.Helper()

	command := []string{"db:execute", sql}
	if service != "db" {
		command = []string{"db:execute", "--service", service, sql}
	}

	var out string
	var err error
	attempts := 0
	started := time.Now()
	deadline := started.Add(dbReadyTimeout)
	// 1130 gets a budget rather than an instant verdict. See refusesTheHost.
	refusalDeadline := started.Add(refusalGrace)
	for {
		attempts++
		out, err = p.tryRun(time.Minute, command...)
		if err == nil || time.Now().After(deadline) {
			break
		}
		if refusesTheHost(out) && time.Now().After(refusalDeadline) {
			break
		}
		time.Sleep(5 * time.Second)
	}
	if err != nil {
		p.t.Fatalf("db:execute %q against %s never succeeded in %s over %d attempts: %v\n%s%s",
			sql, service, time.Since(started).Round(time.Second), attempts, err, out, p.databaseDiagnosis(service))
	}
	return out
}

// refusalGrace is how long 1130 is allowed to be wrong about itself.
//
// It used to be zero: the first 1130 ended the wait, which is why a failure
// reads "never succeeded in 0s over 1 attempts". The reasoning was sound as far
// as it went — the account is created once, while the data directory is
// initialised, so a server that is already answering has decided — and it saved
// five minutes and 61 identical attempts on a real grant defect, twice.
//
// What it did not allow for is a server that answers *before* it has finished
// deciding. That case has not been demonstrated: it does not reproduce on a
// developer machine, where the database is ready in a tenth of a second, and the
// three tests it hits are green on every pull-request run and red on the master
// run of the same tree, twice — a difference that is about the machine, not the
// code.
//
// So this is a trade, made deliberately and with the evidence stated: if 1130 is
// permanent the suite now spends this long instead of nothing, and if it is the
// initialisation window it survives. Being wrong costs a minute; the setting it
// replaces cost two red runs on master and an afternoon of guessing.
const refusalGrace = time.Minute

// refusesTheHost reports MariaDB answering 1130, which is the server running and
// refusing the client's address outright — no grant covers it.
func refusesTheHost(out string) bool {
	return strings.Contains(out, "ERROR 1130")
}

// databaseDiagnosis is what the runner is asked for before it is thrown away.
//
// A hosted runner is disposable, so the container that misbehaved is gone by the
// time anyone reads the failure — and the one thing that separates the two
// candidate causes is in its log: a data directory that was initialised prints
// the initialisation banner, one that was reused goes straight to "ready for
// connections" and creates no accounts. Twice now that log was the missing piece.
func (p *project) databaseDiagnosis(service string) string {
	logs, err := p.tryRun(time.Minute, "logs", "-s", service)
	if err != nil {
		return "\n(could not read the " + service + " log: " + err.Error() + ")"
	}

	lines := strings.Split(strings.TrimRight(logs, "\n"), "\n")

	// Both ends, and the first one is the point.
	//
	// This printed the last 40 lines only, which could never answer the
	// question the comment above says it exists for: whether the data
	// directory was initialised. That banner is written at the very start of
	// the log, so tailing it hid exactly the evidence it was meant to show —
	// and a reader then concludes "no banner, so the directory was reused",
	// which is a conclusion drawn from a window the banner cannot appear in.
	// That mistake was made on this suite, from these logs.
	const head, tail = 25, 25
	if len(lines) <= head+tail {
		return "\n--- the whole " + service + " log (" + strconv.Itoa(len(lines)) + " lines) ---\n" + strings.Join(lines, "\n")
	}

	return "\n--- first " + strconv.Itoa(head) + " lines of the " + service + " log (where initialisation is decided) ---\n" +
		strings.Join(lines[:head], "\n") +
		"\n--- " + strconv.Itoa(len(lines)-head-tail) + " lines omitted ---\n" +
		"--- last " + strconv.Itoa(tail) + " lines ---\n" +
		strings.Join(lines[len(lines)-tail:], "\n")
}

// freshTable creates a table, replacing one an earlier run may have left.
//
// Volumes are named after the project, so a run that was interrupted — killed
// mid-test, or measuring a deliberately broken binary — leaves its database
// behind under the name the next run will use. Without this the next run fails
// with "table already exists", which says nothing about the code and everything
// about last time.
func (p *project) freshTable(name, definition string) {
	p.t.Helper()
	p.query("DROP TABLE IF EXISTS " + name)
	p.query("CREATE TABLE " + name + " " + definition)
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
