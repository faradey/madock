package logrotate

import (
	"strings"
	"testing"
)

// The order inside the script is the whole of its correctness, and it is easy
// to get wrong in a way that reads fine: truncating before copying loses
// everything the file held, and shifting the numbered copies after the copy
// overwrites `.1` with itself.
func TestTheScriptCopiesBeforeItTruncates(t *testing.T) {
	script := Plan{MaxSize: 1 << 20, Keep: 3}.Script()

	copyAt := strings.Index(script, `cp "$f" "$f.1"`)
	truncateAt := strings.Index(script, `: > "$f"`)
	shiftAt := strings.Index(script, `mv "$f.$prev" "$f.$i"`)

	if copyAt < 0 || truncateAt < 0 || shiftAt < 0 {
		t.Fatalf("the script lost one of its three steps:\n%s", script)
	}
	if !(shiftAt < copyAt && copyAt < truncateAt) {
		t.Errorf("the steps are out of order — shift %d, copy %d, truncate %d:\n%s",
			shiftAt, copyAt, truncateAt, script)
	}
}

// Truncation in place, never deletion. The writer holds the inode open, so
// removing the file would leave nginx writing into space nobody can read until
// the container restarts — the same disappearance this whole feature exists to
// stop, reached from the other side.
func TestTheScriptNeverDeletesTheLiveFile(t *testing.T) {
	script := Plan{MaxSize: 1 << 20, Keep: 3}.Script()

	if strings.Contains(script, `rm "$f"`) || strings.Contains(script, "rm -f \"$f\"") {
		t.Errorf("the live log is deleted rather than truncated:\n%s", script)
	}
}

// The size a project configured has to reach the comparison, or rotation
// happens at some number nobody chose.
func TestTheConfiguredSizeReachesTheScript(t *testing.T) {
	script := Plan{MaxSize: 52428800, Keep: 7}.Script()

	if !strings.Contains(script, "52428800") {
		t.Errorf("the threshold is missing from the script:\n%s", script)
	}
	if !strings.Contains(script, "i=7") {
		t.Errorf("the number of copies to keep is missing:\n%s", script)
	}
}

// Keep must never be zero: the loop would shift nothing and `cp` would write
// `.1` over a copy that is still the newest, so the rotation would silently
// keep exactly one generation whatever was asked for.
func TestKeepIsAtLeastOne(t *testing.T) {
	if script := (Plan{MaxSize: 100, Keep: 0}).Script(); !strings.Contains(script, "i=1") {
		t.Errorf("keep=0 did not fall back to one copy:\n%s", script)
	}
}

func TestParseSizeReadsWhatPeopleWrite(t *testing.T) {
	for value, want := range map[string]int64{
		"50M":  50 << 20,
		"50MB": 50 << 20,
		"1G":   1 << 30,
		"512K": 512 << 10,
		"1024": 1024,
	} {
		got, err := ParseSize(value)
		if err != nil {
			t.Errorf("%q: %v", value, err)
			continue
		}
		if got != want {
			t.Errorf("%q read as %d, want %d", value, got, want)
		}
	}
}

// An unreadable size is an error, not a default. A default here means "rotate
// later than you asked", which is silent and is discovered as a full disk.
func TestParseSizeRefusesWhatItCannotRead(t *testing.T) {
	for _, value := range []string{"", "big", "-5M", "0"} {
		if _, err := ParseSize(value); err == nil {
			t.Errorf("%q was accepted as a size", value)
		}
	}
}
