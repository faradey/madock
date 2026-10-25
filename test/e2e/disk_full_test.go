//go:build e2e

package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestAFullDiskIsAFailureAndTheProjectSurvivesIt — the project's directory
// fills up in the middle of a composer install. The command has to fail and
// say why (not report success over a half-written vendor/), the running
// containers have to stay up, and once space is freed the same install has to
// finish. The directory is a small loop filesystem mounted over the project,
// so filling it touches nothing else on the machine.
func TestAFullDiskIsAFailureAndTheProjectSurvivesIt(t *testing.T) {
	if out, err := exec.Command("sudo", "-n", "true").CombinedOutput(); err != nil {
		t.Skipf("needs passwordless sudo for a loop mount: %v %s", err, out)
	}
	p := newProject(t, "e2ediskfull")

	img := filepath.Join(t.TempDir(), "small.img")
	sudo(t, "truncate", "-s", "256M", img)
	sudo(t, "mkfs.ext4", "-q", "-F", img)
	sudo(t, "mount", "-o", "loop", img, p.runDir)
	t.Cleanup(func() { _ = exec.Command("sudo", "-n", "umount", "-l", p.runDir).Run() })
	sudo(t, "chown", strconv.Itoa(os.Getuid())+":"+strconv.Itoa(os.Getgid()), p.runDir)

	p.run(setupTimeout, "setup", "-y", "--platform=custom", "--language=php", "--hosts=e2ediskfull.test")
	p.run(20*time.Minute, "start")
	// The AWS SDK: tens of megabytes unpacked, no advisory for Composer to
	// block it on — a package Composer refuses would fail for that reason
	// and test nothing about the disk.
	if err := os.WriteFile(filepath.Join(p.runDir, "composer.json"), []byte(`{"require": {"aws/aws-sdk-php": "^3.300"}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Leave a few megabytes: enough to start, far too little for the SDK.
	filler := filepath.Join(p.runDir, "filler.bin")
	fill(t, filler, p.runDir, 4<<20)

	out, err := p.tryRun(15*time.Minute, "composer", "install", "--no-interaction")
	if err == nil {
		t.Fatalf("composer install on a full disk reported success:\n%s", out)
	}
	// Composer says it one of two ways: an extraction fails with the
	// kernel's "No space left on device", a download with curl's error 23,
	// "Failure writing output to destination" — measured: the second, here.
	lower := strings.ToLower(out)
	if !strings.Contains(lower, "no space left on device") && !strings.Contains(lower, "failure writing output") {
		t.Errorf("the output does not say the disk is full:\n%s", firstLines(out, 80))
	}
	requireContains(t, p.run(time.Minute, "status"), "php running", "the containers after the disk filled")

	if err := os.Remove(filler); err != nil {
		t.Fatal(err)
	}
	p.run(15*time.Minute, "composer", "install", "--no-interaction")
	requireFile(t, filepath.Join(p.runDir, "vendor", "aws", "aws-sdk-php", "composer.json"),
		"the SDK, installed once space was freed")
}

func sudo(t *testing.T, args ...string) {
	t.Helper()
	if out, err := exec.Command("sudo", append([]string{"-n"}, args...)...).CombinedOutput(); err != nil {
		t.Fatalf("sudo %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

// fill writes a file into dir until about leave bytes of the filesystem are
// free.
func fill(t *testing.T, path, dir string, leave int64) {
	t.Helper()
	out, err := exec.Command("df", "--output=avail", "-B1", dir).Output()
	if err != nil {
		t.Fatalf("df %s: %v", dir, err)
	}
	fields := strings.Fields(string(out))
	avail, err := strconv.ParseInt(fields[len(fields)-1], 10, 64)
	if err != nil {
		t.Fatalf("df %s: %q", dir, out)
	}
	if avail <= leave {
		return
	}
	if out, err := exec.Command("fallocate", "-l", strconv.FormatInt(avail-leave, 10), path).CombinedOutput(); err != nil {
		t.Fatalf("fallocate: %v\n%s", err, out)
	}
}
