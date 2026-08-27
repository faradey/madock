package embedded

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"testing/fstest"
)

// The defect this exists for: the installation and the working copy are the
// same directory in a source install, so extracting a build-time snapshot over
// it reverts whatever was edited since the binary was built. Measured — three
// edited templates disappeared mid-session, and a test then passed against the
// reverted files.
func TestExtractIfNeeded_LeavesASourceCheckoutAlone(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MADOCK_EXEC_DIR", dir)

	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// The signal is the embed declaration in the template tree, not go.mod —
	// see isSourceCheckout for why the module file was the wrong question.
	if err := os.MkdirAll(filepath.Join(dir, "docker"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(sourceSentinel)), []byte("package dockerassets\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	edited := filepath.Join(dir, "docker", "snippets", "probe.yml")
	if err := os.MkdirAll(filepath.Dir(edited), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(edited, []byte("edited by hand\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	DockerFS = fstest.MapFS{
		"snippets/probe.yml": {Data: []byte("from the binary\n")},
	}
	t.Cleanup(func() { DockerFS = nil })

	ExtractIfNeeded("9.9.9")

	body, err := os.ReadFile(edited)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "edited by hand\n" {
		t.Fatalf("the working copy was overwritten by the embedded snapshot: %q", string(body))
	}

	if _, err := os.Stat(filepath.Join(dir, ".embedded_version")); err == nil {
		t.Error("a source checkout should not be stamped with an embedded version either")
	}
}

// The other half of the same question, and the one go.mod got wrong: an
// installation can be a Go module and still have no other way to get its
// templates. madock-pro is a clone with go.mod at its root, but docker/ is in
// its .gitignore on purpose — the assets belong to the imported module and are
// extracted at runtime. Testing go.mod therefore skipped extraction in the one
// installation where nothing else delivers the tree, and it froze: measured
// 2026-08-17, .embedded_version said 3.6.7 while the module was 3.9.3, and 47
// templates were still in the pre-3.9.1 syntax. A customer install, being a
// bare binary, was fine — so the breakage was confined to the installation pro
// is developed and tested on.
func TestExtractIfNeeded_FillsAModuleThatDoesNotShipItsTemplates(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MADOCK_EXEC_DIR", dir)

	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// What an earlier binary left behind, with the stamp to match.
	stale := filepath.Join(dir, "docker", "snippets", "probe.yml")
	if err := os.MkdirAll(filepath.Dir(stale), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stale, []byte("from an old binary\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".embedded_version"), []byte("3.6.7"), 0o644); err != nil {
		t.Fatal(err)
	}

	DockerFS = fstest.MapFS{
		"snippets/probe.yml": {Data: []byte("from the binary\n")},
	}
	t.Cleanup(func() { DockerFS = nil })

	ExtractIfNeeded("9.9.9")

	body, err := os.ReadFile(stale)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "from the binary\n" {
		t.Fatalf("the stale tree was left in place: %q", string(body))
	}

	stamp, err := os.ReadFile(filepath.Join(dir, ".embedded_version"))
	if err != nil || string(stamp) != "9.9.9" {
		t.Fatalf("the version stamp was not updated: %q, %v", string(stamp), err)
	}
}

// The defect this exists for, measured on `shopify-e2e` on 2026-08-27: one file
// in the tree was owned by root, `fs.WalkDir` reads a non-nil answer from the
// callback as "stop", and `extractFS` returned `os.WriteFile`'s result — so the
// extraction ended on that file and 107 templates never reached the disk, in
// silence. Everything after `w` in that directory was missing, which is the
// shape a lexical walk leaves behind.
func TestExtractIfNeeded_WritesTheRestPastAFileItCannotWrite(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MADOCK_EXEC_DIR", dir)

	DockerFS = fstest.MapFS{
		"a/first.yml":   {Data: []byte("first\n")},
		"m/blocked.yml": {Data: []byte("blocked\n")},
		"z/last.yml":    {Data: []byte("last\n")},
	}
	t.Cleanup(func() { DockerFS = nil })

	blockWrites(t, filepath.Join(dir, "docker", "m", "blocked.yml"))

	ExtractIfNeeded("9.9.9")

	// Before the obstacle, so it landed even when the walk stopped.
	if body, err := os.ReadFile(filepath.Join(dir, "docker", "a", "first.yml")); err != nil || string(body) != "first\n" {
		t.Fatalf("the file before the obstacle is wrong: %q, %v", string(body), err)
	}
	// After it, which is the whole question.
	if body, err := os.ReadFile(filepath.Join(dir, "docker", "z", "last.yml")); err != nil || string(body) != "last\n" {
		t.Fatalf("extraction stopped at the file it could not write, so everything after it is missing: %q, %v", string(body), err)
	}
}

// An extraction that did not finish must not declare the installation current:
// ExtractIfNeeded returns early when the marker equals the running version, so a
// stamp on a failed run means nothing ever tries again and the tree stays
// two-thirds written for the life of that version.
func TestExtractIfNeeded_DoesNotStampAVersionItCouldNotFinish(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MADOCK_EXEC_DIR", dir)

	DockerFS = fstest.MapFS{"m/blocked.yml": {Data: []byte("blocked\n")}}
	t.Cleanup(func() { DockerFS = nil })

	blockWrites(t, filepath.Join(dir, "docker", "m", "blocked.yml"))

	ExtractIfNeeded("9.9.9")

	if _, err := os.Stat(filepath.Join(dir, ".embedded_version")); err == nil {
		t.Error("an extraction that failed stamped its version, so nothing will retry it")
	}
}

// The second-order defect, and the expensive one: `removeWithdrawn` deletes what
// the previous manifest names and this run did not write. A truncated run reads
// its own gap as "the release withdrew these" — so a single unwritable file does
// not merely leave templates unwritten on a fresh installation, it strips an
// upgraded one of templates that are still shipped and were working an hour ago.
func TestExtractIfNeeded_DeletesNothingWhenItCouldNotFinish(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MADOCK_EXEC_DIR", dir)

	DockerFS = fstest.MapFS{
		"m/blocked.yml": {Data: []byte("blocked\n")},
		"z/last.yml":    {Data: []byte("last\n")},
	}
	t.Cleanup(func() { DockerFS = nil })

	// What a healthy extraction left behind: the file, and the manifest saying
	// this mechanism owns it.
	survivor := filepath.Join(dir, "docker", "z", "last.yml")
	mustWrite(t, survivor, "last\n")
	manifest := filepath.Join(dir, manifestFile)
	mustWrite(t, manifest, "docker/m/blocked.yml\ndocker/z/last.yml\n")

	blockWrites(t, filepath.Join(dir, "docker", "m", "blocked.yml"))

	ExtractIfNeeded("9.9.9")

	if _, err := os.Stat(survivor); err != nil {
		t.Fatalf("a template that is still shipped was deleted because this run never reached it: %v", err)
	}

	body, err := os.ReadFile(manifest)
	if err != nil || string(body) != "docker/m/blocked.yml\ndocker/z/last.yml\n" {
		t.Fatalf("the incomplete run published its gap as the manifest, so the next one inherits it: %q, %v", string(body), err)
	}
}

// One run under MADOCK_USER=root leaves root-owned files in the installation and
// every later extraction as the ordinary user is refused on them for good.
// Unlink permission belongs to the directory, so the file can be replaced
// outright — which is how a tree mined that way repairs itself instead of
// needing a chown nobody knows to run.
func TestExtractIfNeeded_ReplacesATemplateItCannotWriteInPlace(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root writes through the mode bits, so there is nothing to refuse")
	}

	dir := t.TempDir()
	t.Setenv("MADOCK_EXEC_DIR", dir)

	DockerFS = fstest.MapFS{"general/probe.yml": {Data: []byte("from the binary\n")}}
	t.Cleanup(func() { DockerFS = nil })

	locked := filepath.Join(dir, "docker", "general", "probe.yml")
	mustWrite(t, locked, "left by a run as another user\n")
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(locked, 0o644) })

	ExtractIfNeeded("9.9.9")

	body, err := os.ReadFile(locked)
	if err != nil || string(body) != "from the binary\n" {
		t.Fatalf("the unwritable template was not replaced: %q, %v", string(body), err)
	}
	if stamp, err := os.ReadFile(filepath.Join(dir, ".embedded_version")); err != nil || string(stamp) != "9.9.9" {
		t.Fatalf("a run that repaired itself should still count as finished: %q, %v", string(stamp), err)
	}
}

// The failure has to be said out loud — the installation looks healthy
// otherwise — and on stderr, because a warning on stdout makes every `--json`
// command unparseable.
func TestExtractFailuresAreNamedOnStderr(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MADOCK_EXEC_DIR", dir)

	DockerFS = fstest.MapFS{"m/blocked.yml": {Data: []byte("blocked\n")}}
	t.Cleanup(func() { DockerFS = nil })

	blockWrites(t, filepath.Join(dir, "docker", "m", "blocked.yml"))

	stdout, stderr := capture(t, func() { ExtractIfNeeded("9.9.9") })

	if stdout != "" {
		t.Errorf("the warning went to stdout, so any --json output is now unparseable:\n%s", stdout)
	}
	if !strings.Contains(stderr, "docker/m/blocked.yml") {
		t.Errorf("the file that could not be written was not named:\n%s", stderr)
	}
	if !strings.Contains(stderr, "tries again") {
		t.Errorf("nothing said the next command retries, which is the one thing to do about it:\n%s", stderr)
	}
}

func TestReportFailuresCountsAndTrims(t *testing.T) {
	failures := []ExtractFailure{
		{Path: "docker/snippets/e.yml", Err: &fs.PathError{Op: "open", Path: "/opt/madock/docker/snippets/e.yml", Err: fs.ErrPermission}},
		{Path: "docker/snippets/a.yml", Err: fs.ErrPermission},
		{Path: "docker/snippets/b.yml", Err: fs.ErrPermission},
		{Path: "docker/snippets/c.yml", Err: fs.ErrPermission},
		{Path: "docker/snippets/d.yml", Err: fs.ErrPermission},
	}

	var out bytes.Buffer
	reportFailures(&out, failures, "/opt/madock")
	body := out.String()

	if !strings.Contains(body, "5 of madock's own templates") {
		t.Errorf("the count is wrong:\n%s", body)
	}
	// Sorted, so the same broken tree reads the same way twice.
	if !strings.Contains(body, "  docker/snippets/a.yml: permission denied") {
		t.Errorf("the first failure is not named, or not sorted:\n%s", body)
	}
	if !strings.Contains(body, "and 2 more") {
		t.Errorf("the remainder was not counted:\n%s", body)
	}
	// The wrapper already carries the path; printing it twice on one line is
	// how a message stops being read.
	if strings.Contains(body, "/opt/madock/docker/snippets/e.yml") {
		t.Errorf("the path is repeated inside the reason:\n%s", body)
	}
}

// blockWrites makes one path impossible to write to for any user, root
// included: a directory with something in it. Refusing by mode instead would
// mean the test proves nothing in a container running as root — which is where
// the fault was found in the first place.
func blockWrites(t *testing.T, target string) {
	t.Helper()

	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "occupant"), []byte("in the way\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// The sentinel is a path, so a rename would turn the guard off silently and the
// reverted-edits defect would come back with nothing failing. This is the only
// test that knows where the repository root is, and that is the point.
func TestSourceSentinelIsWhereTheGuardExpectsIt(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Skip("no caller information")
	}
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(sourceSentinel))); err != nil {
		t.Fatalf("%s is gone from the repository, so isSourceCheckout now answers false "+
			"for madock's own tree and a development binary will revert edited templates again: %v",
			sourceSentinel, err)
	}
}

// A binary installation has nothing but the binary, so the templates have to
// come out of it — that path must keep working.
func TestExtractIfNeeded_StillFillsABinaryInstallation(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MADOCK_EXEC_DIR", dir)

	DockerFS = fstest.MapFS{
		"snippets/probe.yml": {Data: []byte("from the binary\n")},
	}
	t.Cleanup(func() { DockerFS = nil })

	ExtractIfNeeded("9.9.9")

	body, err := os.ReadFile(filepath.Join(dir, "docker", "snippets", "probe.yml"))
	if err != nil {
		t.Fatalf("the templates were not extracted: %v", err)
	}
	if string(body) != "from the binary\n" {
		t.Fatalf("unexpected content: %q", string(body))
	}

	stamp, err := os.ReadFile(filepath.Join(dir, ".embedded_version"))
	if err != nil || string(stamp) != "9.9.9" {
		t.Fatalf("the version stamp was not written: %q, %v", string(stamp), err)
	}
}
