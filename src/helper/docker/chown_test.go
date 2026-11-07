package docker

import (
	"strings"
	"testing"
)

// TestChownWorkdirSkipsMadock guards the chown that reclaims ownership of the
// project code. Every container mounts `.madock` read-only over the code, so a
// plain `chown -R` of the workdir stops on "Read-only file system" — which is
// what broke start --with-chown. The command has to step around `.madock` and
// chown the rest.
//
// The behaviour is covered end to end by TestStartWithChownTakesOwnershipBack;
// this holds the command shape so a change to it fails here first.
func TestChownWorkdirSkipsMadock(t *testing.T) {
	cmd := ChownWorkdirSkippingMadock("1000", "1001", "/var/www/html")

	if !strings.Contains(cmd, "-name .madock -prune") {
		t.Errorf("chown does not step around .madock, so it fails on the read-only mount:\n%s", cmd)
	}
	if !strings.Contains(cmd, "chown 1000:1001") {
		t.Errorf("chown does not apply the given uid:gid:\n%s", cmd)
	}
	if !strings.Contains(cmd, "/var/www/html") {
		t.Errorf("chown does not target the workdir:\n%s", cmd)
	}
}
