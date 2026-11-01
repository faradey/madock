package docker

import (
	"errors"
	"strings"
	"testing"
)

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

// What madock actually asks compose for.
//
// The distinction this guards is the one the 2021 commit made and the code lost:
// a rebuild goes and looks for a newer image, an ordinary start only needs the
// image present. Getting it backwards is invisible — both spellings work, one
// of them just talks to the registry about images the machine already has, on
// every first start of every installation.
func TestPullAsksForAPolicyOnlyWhenItShould(t *testing.T) {
	files := []string{"compose", "-f", "docker-compose.yml"}

	cases := []struct {
		name      string
		refresh   bool
		supported bool
		quiet     bool
		want      []string
	}{
		{
			name: "an ordinary start on a compose that understands policies",
			want: []string{"compose", "-f", "docker-compose.yml", "pull", "--policy", "missing"},
		},
		{
			name:    "a rebuild, which is the one case that means to look for a newer image",
			refresh: true,
			want:    []string{"compose", "-f", "docker-compose.yml", "pull"},
		},
		{
			name: "compose too old to be told, where the old behaviour has to stand",
			want: []string{"compose", "-f", "docker-compose.yml", "pull"},
		},
		{
			name:  "quiet comes last, and does not displace the policy",
			quiet: true,
			want:  []string{"compose", "-f", "docker-compose.yml", "pull", "--policy", "missing", "--quiet"},
		},
	}

	// Only the first, second and fourth run against a compose that supports it;
	// the third is the whole point of the flag being conditional.
	cases[0].supported = true
	cases[1].supported = true
	cases[3].supported = true

	for _, c := range cases {
		got := pullArgs(files, c.refresh, c.supported, c.quiet)
		if len(got) != len(c.want) {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("%s: got %v, want %v", c.name, got, c.want)
				break
			}
		}
	}

	// The caller's slice is not to be scribbled on: dockerComposePull is handed
	// a literal at three call sites today, and an append that reuses the backing
	// array would leak "pull" into whatever else shared it.
	if len(files) != 3 {
		t.Errorf("pullArgs modified the slice it was given: %v", files)
	}
}

// A rebuild for a security fix has to reach the package layer: without
// --no-cache the cached `apt-get install` layer is reused and the old package
// stays, and without --pull the base image is the one already on the machine.
func TestBuildWithoutCacheSkipsTheCacheAndPullsTheBase(t *testing.T) {
	files := []string{"compose", "-f", "docker-compose.yml"}

	got := buildWithoutCacheArgs(files, false)
	want := []string{"compose", "-f", "docker-compose.yml", "build", "--no-cache", "--pull"}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("got %v, want %v", got, want)
	}

	got = buildWithoutCacheArgs(files, true)
	if got[len(got)-1] != "--quiet" {
		t.Errorf("quiet: got %v, want --quiet last", got)
	}

	if len(files) != 3 {
		t.Errorf("buildWithoutCacheArgs modified the slice it was given: %v", files)
	}
}

// A rebuild reaches the pull with the containers already down, and with the
// shared proxy down when no other project runs. A registry that does not
// answer must not leave them there when the images are on the machine.
func TestARefreshThatCannotReachTheRegistryUsesTheLocalImages(t *testing.T) {
	var calls [][]string
	warned := ""
	run := func(args []string) error {
		calls = append(calls, args)
		if len(calls) == 1 {
			return errors.New("timeout")
		}
		return nil
	}
	err := pullWithFallback([]string{"compose", "-f", "x.yml"}, true, true, false, run, func(m string) { warned = m })
	if err != nil {
		t.Fatalf("the fallback found the images and still failed: %v", err)
	}
	if len(calls) != 2 || !strings.Contains(strings.Join(calls[1], " "), "--policy missing") {
		t.Fatalf("calls: %v", calls)
	}
	if !strings.Contains(warned, "timeout") {
		t.Fatalf("nothing said about the registry: %q", warned)
	}
}

// A missing image still fails: there is nothing on the machine to run.
func TestAMissingImageStillFails(t *testing.T) {
	run := func(args []string) error { return errors.New("no route") }
	if err := pullWithFallback([]string{"compose"}, true, true, false, run, func(string) {}); err == nil {
		t.Fatal("both pulls failed and the error was swallowed")
	}
	// Start, not rebuild, has nothing to fall back to: one call, its error.
	calls := 0
	once := func(args []string) error { calls++; return errors.New("no route") }
	if err := pullWithFallback([]string{"compose"}, false, true, false, once, func(string) {}); err == nil || calls != 1 {
		t.Fatalf("err %v after %d calls", err, calls)
	}
}
