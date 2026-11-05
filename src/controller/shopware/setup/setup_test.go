package setup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// fakeComposer behaves as composer does when a download fails: it refuses a
// directory that is not empty, and a failed attempt leaves a file behind. It fails
// the first failures calls and writes one line per call to the file it returns.
func fakeComposer(t *testing.T, failures int) (binDir, workDir, calls string) {
	t.Helper()

	binDir, workDir = t.TempDir(), t.TempDir()
	calls = filepath.Join(binDir, "calls")
	script := fmt.Sprintf(`#!/bin/sh
echo "$@" >> %[1]s
dir="$3"
if [ -n "$(ls -A "$dir")" ]; then echo "Project directory is not empty" >&2; exit 1; fi
if [ "$(wc -l < %[1]s)" -le %[2]d ]; then touch "$dir/partial"; exit 1; fi
touch "$dir/composer.json"
`, calls, failures)
	if err := os.WriteFile(filepath.Join(binDir, "composer"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(workDir, "download"), 0o755); err != nil {
		t.Fatal(err)
	}
	return binDir, workDir, calls
}

func runCreateProject(t *testing.T, binDir, workDir string) (string, error) {
	t.Helper()
	cmd := exec.Command("/bin/bash", "-c", createProjectScript("6.7.14.1", "./download", 0))
	cmd.Dir = workDir
	cmd.Env = []string{"PATH=" + binDir + ":/usr/bin:/bin"}
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func callCount(t *testing.T, calls string) int {
	t.Helper()
	data, err := os.ReadFile(calls)
	if err != nil {
		t.Fatal(err)
	}
	return strings.Count(string(data), "\n")
}

// One failed download must not end a setup whose rebuild took five minutes, and
// the attempt after it must not trip over what the failure left in the directory.
func TestCreateProjectSurvivesAFailedDownload(t *testing.T) {
	binDir, workDir, calls := fakeComposer(t, 2)

	out, err := runCreateProject(t, binDir, workDir)
	if err != nil {
		t.Fatalf("the third attempt should have succeeded: %v\n%s", err, out)
	}
	if got := callCount(t, calls); got != 3 {
		t.Errorf("composer ran %d times; want 3\n%s", got, out)
	}
	if _, err := os.Stat(filepath.Join(workDir, "download", "composer.json")); err != nil {
		t.Errorf("the project is not in the directory: %v", err)
	}
	if _, err := os.Stat(filepath.Join(workDir, "download", "partial")); err == nil {
		t.Error("what the failed attempt left is still in the directory")
	}
}

// A failure that does not pass still fails the setup, and says what the name
// resolved to, which is the one thing the CI failures did not record.
func TestCreateProjectGivesUpAfterThreeAttempts(t *testing.T) {
	binDir, workDir, calls := fakeComposer(t, 100)

	out, err := runCreateProject(t, binDir, workDir)
	if err == nil {
		t.Fatalf("a download that never works was reported as success:\n%s", out)
	}
	if got := callCount(t, calls); got != 3 {
		t.Errorf("composer ran %d times; want 3\n%s", got, out)
	}
	if !strings.Contains(out, "api.github.com resolves to:") {
		t.Errorf("a failed attempt did not report the address:\n%s", out)
	}
}

// The first attempt that works is the last one.
func TestCreateProjectStopsAtTheFirstSuccess(t *testing.T) {
	binDir, workDir, calls := fakeComposer(t, 0)

	if out, err := runCreateProject(t, binDir, workDir); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if got := callCount(t, calls); got != 1 {
		t.Errorf("composer ran %d times; want 1", got)
	}
}
