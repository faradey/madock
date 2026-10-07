//go:build e2e

package e2e

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestPhpImagesOfDifferentTimezonesShareTheToolchain asks docker whether two
// projects that differ only in their timezone keep one copy of the PHP
// toolchain or two.
//
// The timezone used to open the first RUN of the php header. A layer's cache
// key is its whole command, so the timezone was part of the key of the
// heaviest layer and of everything built on it: two projects on one machine
// with different timezones shared nothing past the ubuntu base. On the madock-pro
// e2e VM on 2026-10-07 two php images of one version and one timezone shared
// 1.615GB of about 1.9GB — that share is what a second timezone threw away.
//
// The witness is the images' layer list, not the rendered Dockerfile: the
// Dockerfile can look right and the cache still not match.
func TestPhpImagesOfDifferentTimezonesShareTheToolchain(t *testing.T) {
	install := newInstallation(t)

	first := install.project("e2etzone")
	first.run(15*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=php",
		"--hosts=e2etzone.test",
	)

	second := install.project("e2etztwo")
	second.run(15*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=php",
		"--hosts=e2etztwo.test",
	)
	second.run(2*time.Minute, "config:set", "-n", "timezone", "-v", "Europe/Kyiv")

	first.run(25*time.Minute, "start")
	second.run(25*time.Minute, "start")

	// The setting itself must still arrive: moving it is only right if the
	// end state is the one it was before.
	if tz := strings.TrimSpace(second.run(2*time.Minute, "cli", "cat", "/etc/timezone")); tz != "Europe/Kyiv" {
		t.Errorf("/etc/timezone in the second project is %q, expected Europe/Kyiv", tz)
	}

	a := imageLayers(t, "madock_e2etzone-php")
	b := imageLayers(t, "madock_e2etztwo-php")

	shared := 0
	for shared < len(a) && shared < len(b) && a[shared] == b[shared] {
		shared++
	}
	t.Logf("layers: %d and %d, shared prefix %d", len(a), len(b), shared)

	// One shared layer is the ubuntu base and nothing else — the old
	// behaviour. The toolchain is the base, the package RUN, the php RUN and
	// the optional-extension RUNs after it, so anything short of five means
	// the timezone is still in front of them.
	if shared < 5 {
		t.Errorf("the two php images share %d layers out of %d — the timezone still sits in front of the toolchain", shared, len(a))
	}
	if shared == len(a) || shared == len(b) {
		t.Errorf("the images are identical — the timezone did not reach the second one")
	}
}

// imageLayers returns an image's layer digests, bottom first.
func imageLayers(t *testing.T, image string) []string {
	t.Helper()

	out, err := exec.Command("docker", "image", "inspect", "--format", "{{json .RootFS.Layers}}", image).CombinedOutput()
	if err != nil {
		t.Fatalf("inspecting %s: %v\n%s", image, err, out)
	}
	var layers []string
	if err := json.Unmarshal(out, &layers); err != nil {
		t.Fatalf("reading the layers of %s (%q): %v", image, out, err)
	}
	return layers
}
