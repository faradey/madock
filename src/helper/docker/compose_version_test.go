package docker

import "testing"

// The boundary is a real date and a real release — compose v2.22.0, 2023-09-21,
// is where `docker compose pull --policy` first appears in the command's own
// reference. Below it the flag is not "ignored", it is an unknown flag and
// compose exits non-zero, so a wrong answer here does not slow madock down, it
// stops it.
func TestPullPolicyIsAskedForNotAssumed(t *testing.T) {
	cases := []struct {
		version   string
		supported bool
		why       string
	}{
		{"2.22.0", true, "the release that added the flag"},
		{"2.22.1", true, "later patch of the same minor"},
		{"2.29.7", true, "a current build"},
		{"3.0.0", true, "a future major must not read as older"},
		{"2.21.0", false, "the release before it"},
		{"2.9.0", false, "a minor that sorts above 2.22 as a string and below it as a number"},
		{"1.29.2", false, "compose v1, which has no such flag and never will"},
		{"", false, "no answer is not an old answer, but it must not enable the flag either"},
	}

	for _, c := range cases {
		got := versionHasPullPolicy(c.version)
		if got != c.supported {
			t.Errorf("versionHasPullPolicy(%q) = %v, want %v — %s", c.version, got, c.supported, c.why)
		}
	}
}
