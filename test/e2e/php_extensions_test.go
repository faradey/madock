//go:build e2e

package e2e

import (
	"strings"
	"testing"
	"time"
)

// TestTheOptionalExtensionsAreActuallyInTheImage asks the interpreter, which is
// the only witness that counts.
//
// The optional block in the PHP Dockerfile installs each package on its own line
// so that one missing package cannot abort the image. Until 2026-09-10 each line
// ended in `|| true`, and that is exactly how a package that never installed
// looked like a package that did: the build was green, the extension was absent,
// and nothing anywhere said so. Measured on the Magento stand — a full rebuild
// exited 0, every service came up, and `php -r extension_loaded('ldap')` printed
// 0 with no file in mods-available and no dpkg entry.
//
// The cause was the layer: the long install above runs `apt-get update` in its
// own RUN, and these lines ran against whatever index that cached layer had left
// behind. Each refreshes the index now, and says out loud what it could not get.
//
// So this test does not read the Dockerfile — a rendered line proves nothing,
// which was the whole lesson. It asks the running container what it loaded.
func TestTheOptionalExtensionsAreActuallyInTheImage(t *testing.T) {
	p := newProject(t, "e2ephpext")
	p.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=php",
		"--hosts=e2ephpext.test",
	)

	p.run(25*time.Minute, "start")

	loaded := p.run(3*time.Minute, "cli", "php", "-m")

	// ldap is the one this test was written for: an application that
	// authenticates against a directory had nowhere to run without it.
	if !containsExtension(loaded, "ldap") {
		t.Errorf("ldap is not loaded in the image — the install line reported success and did nothing:\n%s", loaded)
	}

	// opcache and xmlrpc share the same block and shared the same `|| true`, so
	// they shared the same hole. opcache in particular is not optional in any
	// practical sense: without it every request recompiles.
	if !containsExtension(loaded, "Zend OPcache") && !containsExtension(loaded, "opcache") {
		t.Errorf("opcache is not loaded — the same silent failure, on the extension that costs the most:\n%s", loaded)
	}
}

// containsExtension matches a line of `php -m` output exactly, because a
// substring match would find "ldap" inside another name and report a hole as
// filled — the failure this test exists to catch, wearing a different hat.
func containsExtension(output, name string) bool {
	for _, line := range strings.Split(output, "\n") {
		if strings.EqualFold(strings.TrimSpace(line), name) {
			return true
		}
	}

	return false
}
