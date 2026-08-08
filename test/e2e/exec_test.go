//go:build e2e

package e2e

import (
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestCliRunsAsTheInvokingUser guards the one line that decides whether files
// written inside a container can be edited outside it.
//
//	RUN usermod -u <UID> -o www-data && groupmod -g <GID> -o www-data
//
// The ids come from whoever ran madock, and the `-o` is not a stylistic choice:
// on macOS the primary group is usually 20, which Debian images have already
// given to `dialout`. Without `-o` the image fails to build there and works
// everywhere else — the shape of defect that reaches customers, because it
// cannot be seen from the machine that has the other number.
//
// This runs on Linux, so it cannot see that collision. What it can see is
// whether the mapping happens at all, which is the part that would silently
// leave every file in the project owned by root.
func TestCliRunsAsTheInvokingUser(t *testing.T) {
	p := newProject(t, "e2euser")

	p.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2euser.test",
	)
	p.run(20*time.Minute, "start")

	uid := p.run(2*time.Minute, "cli", "id", "-u")
	if got := lastNumber(uid); got != os.Getuid() {
		t.Errorf("cli runs as uid %d, but madock was invoked by %d:\n%s", got, os.Getuid(), uid)
	}

	gid := p.run(2*time.Minute, "cli", "id", "-g")
	if got := lastNumber(gid); got != os.Getgid() {
		t.Errorf("cli runs as gid %d, but madock was invoked by %d:\n%s", got, os.Getgid(), gid)
	}

	// A file created inside has to be writable outside, which is the whole
	// reason the ids are mapped. Asserting the id alone would pass on an image
	// whose umask makes the result read-only anyway.
	p.run(2*time.Minute, "cli", "touch", "/var/www/html/written-inside")
	written := p.runDir + "/written-inside"
	info, err := os.Stat(written)
	if err != nil {
		t.Fatalf("a file created in the container did not appear in the project directory: %v", err)
	}
	if info.Mode().Perm()&0o200 == 0 {
		t.Errorf("a file created in the container is not writable outside it: %v", info.Mode())
	}
}

// lastNumber returns the final integer in a command's output. `id -u` prints one
// number, but madock may print a line of its own first, and a parser that takes
// the whole output fails on a day when it does.
func lastNumber(out string) int {
	fields := strings.Fields(out)
	for i := len(fields) - 1; i >= 0; i-- {
		if n, err := strconv.Atoi(strings.TrimSpace(fields[i])); err == nil {
			return n
		}
	}
	return -1
}
