// Package logrotate keeps the persisted container logs from growing without end.
//
// The logs live in a volume mounted at /var/log/madock, written by nginx and by
// the database. Nothing inside those images rotates them: the official nginx
// image ships no logrotate, MariaDB rotates nothing of its own, and the crontab
// madock installs runs in the PHP container, which is not where these files are.
// Left alone the access log of a busy site fills a disk, and the first anyone
// hears of it is a machine that has stopped.
//
// The rotation is copy-and-truncate, and that is the one decision worth stating.
// Ordinary rotation renames the file and asks the writer to reopen it — a signal
// this cannot send, because the process is in another container and madock is
// not its parent. Copying the contents aside and truncating the file in place
// leaves the writer's file descriptor pointing at the same inode, so nginx and
// mysqld go on writing without being told anything. The cost is the window
// between the copy and the truncate: a line written in it is lost. That is the
// standard trade for containers, and it is the reason the sizes here are large
// enough that rotation is rare.
package logrotate

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// LogDir is where the services write, inside every container that mounts the
// volume.
const LogDir = "/var/log/madock"

// Plan is what a rotation would do, with the numbers a project configured.
type Plan struct {
	// MaxSize is the size a file has to reach before it is rotated, in bytes.
	MaxSize int64
	// Keep is how many rotated copies survive. The oldest goes when a new one
	// arrives.
	Keep int
}

// Script renders the shell a container runs to rotate its own logs.
//
// POSIX sh and nothing else: it runs in the nginx image and in the MariaDB one,
// and the only tools both are guaranteed to have are the ones in this text. No
// `find -size`, no `stat -c` — busybox and coreutils disagree about both.
//
// The order matters and is the whole of the correctness here: the oldest copy
// is dropped first, the rest shift up, the live file is copied to `.1` and only
// then truncated. Truncating before the copy would lose everything the file
// held; shifting after the copy would overwrite `.1` with itself.
func (p Plan) Script() string {
	keep := p.Keep
	if keep < 1 {
		keep = 1
	}

	return strings.Join([]string{
		"set -e",
		"cd " + LogDir + " 2>/dev/null || exit 0",
		"for f in *.log; do",
		"  [ -f \"$f\" ] || continue",
		// wc -c rather than stat: the two images disagree about stat's flags,
		// and a rotation that fails on one of them is a log that grows for ever
		// on that one.
		"  size=$(wc -c < \"$f\")",
		"  [ \"$size\" -ge " + strconv.FormatInt(p.MaxSize, 10) + " ] || continue",
		"  i=" + strconv.Itoa(keep),
		"  while [ \"$i\" -gt 1 ]; do",
		"    prev=$((i-1))",
		"    [ -f \"$f.$prev\" ] && mv \"$f.$prev\" \"$f.$i\"",
		"    i=$prev",
		"  done",
		"  cp \"$f\" \"$f.1\"",
		// The truncate. `: >` rather than rm: the writer holds this inode open,
		// and deleting the file would leave it writing into space nobody can
		// read until the container restarts — the exact failure this package
		// exists to avoid, arrived at from the other direction.
		"  : > \"$f\"",
		"done",
	}, "\n")
}

// ParseSize reads a size the way a person writes it in configuration: 50M, 1G,
// 512K, or a bare number of bytes.
//
// An unreadable value is an error rather than a default, because the default
// here is "rotate later than you asked" — silent, and discovered as a full disk.
func ParseSize(value string) (int64, error) {
	text := strings.TrimSpace(strings.ToUpper(value))
	if text == "" {
		return 0, fmt.Errorf("empty size")
	}

	multiplier := int64(1)
	switch {
	case strings.HasSuffix(text, "K"), strings.HasSuffix(text, "KB"):
		multiplier = 1 << 10
	case strings.HasSuffix(text, "M"), strings.HasSuffix(text, "MB"):
		multiplier = 1 << 20
	case strings.HasSuffix(text, "G"), strings.HasSuffix(text, "GB"):
		multiplier = 1 << 30
	}
	text = strings.TrimRight(text, "KMGB")

	amount, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
	if err != nil || amount <= 0 {
		return 0, fmt.Errorf("%q is not a size: expected something like 50M or 1G", value)
	}

	return int64(amount * float64(multiplier)), nil
}

// Rotate runs the plan inside one container and reports what the shell said.
//
// Failures are returned rather than fatal for the caller to decide: rotation
// runs after `start`, and a project whose web server is up must not be reported
// as broken because a log file could not be copied.
var Rotate = func(container string, plan Plan) (string, error) {
	cmd := exec.Command("docker", "exec", container, "sh", "-c", plan.Script())
	out, err := cmd.CombinedOutput()

	return string(out), err
}
