//go:build e2e

package e2e

import (
	"strings"
	"testing"
	"time"
)

// TestShopwareCliRefusesUntilItIsEnabled covers the half that costs nothing.
//
// shopware-cli is downloaded into the php image at build time and only when
// shopware/cli/enabled says so, so on a project that has not asked for it the
// exec would fail with "shopware-cli: command not found" — a message about a
// missing binary, which reads as a broken installation rather than as a setting
// nobody turned on. The command answers before reaching docker instead.
//
// No `start` and no `rebuild` here on purpose: the refusal happens before either
// is needed, and a test that built an image to prove it would be paying minutes
// to hide the very thing it is checking.
func TestShopwareCliRefusesUntilItIsEnabled(t *testing.T) {
	p := newProject(t, "e2eswcli")

	// Twenty minutes, not five: the first Shopware setup on a machine pulls the
	// php, nginx and opensearch images, and a clean CI runner is always that
	// machine. Measured here — 22s with the images cached, 390s without, which
	// is how a five-minute budget turns a working feature into a red suite.
	p.run(20*time.Minute, "setup", "-y",
		"--platform=shopware",
		// The version is asked for even under -y, so it has to be named: without
		// it setup stops at the prompt and the test dies on EOF. Nothing is
		// downloaded — no -d and no -i — so this only decides what the stack is
		// configured for.
		"--platform-version=6.7.13.0",
		"--php=8.3",
		"--hosts=e2eswcli.test",
	)

	out, err := p.tryRun(2*time.Minute, "sw:cli", "--version")
	if err == nil {
		t.Fatalf("sw:cli ran on a project that never enabled it:\n%s", out)
	}

	// The key by name, because the whole point is that the reader does not have
	// to know it already. And the rebuild, because setting the key alone changes
	// nothing until the image is built again.
	requireContains(t, out, "shopware/cli/enabled", "the refusal naming the config key")
	requireContains(t, out, "rebuild", "the refusal naming the rebuild")

	// Not the message docker would produce. If this ever appears, the check in
	// front of the exec has stopped running and the command reached the
	// container.
	if strings.Contains(out, "command not found") {
		t.Errorf("the refusal came from the container rather than from madock:\n%s", out)
	}
}

// TestShopwareCliIsInstalledIntoTheImage is the other half, and it builds a
// Shopware php image — minutes, and a download from GitHub — so it sits behind
// the same opt-in as the tests that install a real store.
//
// It deliberately does **not** install Shopware. The tooling is part of the
// image rather than of the project, which is the whole design: `setup` without
// --download and --install configures the stack and nothing more, so this proves
// the image half while Shopware's own installer is unusable — and it has been
// since 2026-08-14, because Shopware pins exact dependency versions that
// composer refuses over advisories. A test that needed a store would be
// permanently skipped instead.
func TestShopwareCliIsInstalledIntoTheImage(t *testing.T) {
	if !platformTestsEnabled() {
		t.Skip("builds a Shopware php image: ./test/e2e/e2e.sh run --platforms -run TestShopwareCliIsInstalled")
	}

	p := newProject(t, "e2eswcliimg")

	// Twenty minutes, not five: the first Shopware setup on a machine pulls the
	// php, nginx and opensearch images, and a clean CI runner is always that
	// machine. Measured here — 22s with the images cached, 390s without, which
	// is how a five-minute budget turns a working feature into a red suite.
	p.run(20*time.Minute, "setup", "-y",
		"--platform=shopware",
		"--platform-version=6.7.13.0",
		"--php=8.3",
		"--hosts=e2eswcliimg.test",
	)

	// Through service:enable rather than config:set, because that is the name a
	// person reaches for and it has to be registered: madock's idea of a service
	// includes tooling baked into an image — n98magerun, mftf, ioncube and
	// xdebug are all in the same registry, and none of them is a container.
	p.run(2*time.Minute, "service:enable", "shopware-cli")

	listed := p.run(2*time.Minute, "config:list")
	requireContains(t, listed, "shopware/cli/enabled", "the key service:enable was supposed to set")

	// The image is built here. Generous: it installs a PHP stack and downloads
	// the release from GitHub.
	p.run(30*time.Minute, "rebuild")

	version := p.run(3*time.Minute, "sw:cli", "--version")
	requireContains(t, version, "0.17.3", "the version pinned by shopware/cli/version")

	// It runs as the project's user and can read the project. `extension` is the
	// command family the tool exists for here — validating and zipping a plugin
	// — so its presence is what makes the image useful rather than merely
	// populated.
	help := p.run(3*time.Minute, "sw:cli", "--help")
	requireContains(t, help, "extension", "the command family the tool was installed for")
}
