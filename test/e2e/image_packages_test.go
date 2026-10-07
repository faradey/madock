//go:build e2e

package e2e

import (
	"strings"
	"testing"
	"time"
)

// TestExtraPackagesReachTheImage asks the running container whether what a
// project added through php/packages/extra and php/extensions/extra is there.
//
// The rendered Dockerfile is not the witness — the optional-extension block
// rendered correctly for months while installing nothing. And one of the names
// does not exist on purpose: an extra the index lacks has to be reported and
// skipped, never stop the build, because a rebuild that stops halfway on a
// server leaves the site down.
func TestExtraPackagesReachTheImage(t *testing.T) {
	p := newProject(t, "e2epkgs")
	p.run(15*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=php",
		"--hosts=e2epkgs.test",
	)
	p.run(2*time.Minute, "config:set", "-n", "php/packages/extra", "-v", "tree no-such-package-madock")
	p.run(2*time.Minute, "config:set", "-n", "php/extensions/extra", "-v", "gmp")

	p.run(25*time.Minute, "start")

	if out := p.run(3*time.Minute, "cli", "tree", "--version"); !strings.Contains(out, "tree v") {
		t.Errorf("tree from php/packages/extra is not in the image:\n%s", out)
	}
	if loaded := p.run(3*time.Minute, "cli", "php", "-m"); !containsExtension(loaded, "gmp") {
		t.Errorf("gmp from php/extensions/extra is not loaded:\n%s", loaded)
	}
}
