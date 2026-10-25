//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestAnAcceptedAdvisoryLetsComposerInstallPastIt is the mechanism setup
// --allow-advisory relies on, run through the Composer madock's images carry:
// league/flysystem 2.x is blocked by PKSA-w9tt-7782-78jx (every Magento below
// 2.4.8 needs it), and naming that one advisory in the project's composer.json
// as audit.ignore lets the install through. If a Composer release stopped
// reading audit.ignore for blocking, the Magento download with
// --allow-advisory would fail again and this is where it would show.
func TestAnAcceptedAdvisoryLetsComposerInstallPastIt(t *testing.T) {
	p := newProject(t, "e2eadvisory")
	p.run(setupTimeout, "setup", "-y", "--platform=custom", "--language=php", "--hosts=e2eadvisory.test")
	p.run(20*time.Minute, "start")
	if err := os.WriteFile(filepath.Join(p.runDir, "composer.json"), []byte(`{"require": {"league/flysystem": "2.5.0"}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := p.tryRun(10*time.Minute, "composer", "install", "--no-interaction")
	if err == nil {
		t.Fatalf("league/flysystem 2.5.0 installed with its advisory not accepted — the block this rests on is gone:\n%s", out)
	}
	requireContains(t, out, "PKSA-w9tt-7782-78jx", "the advisory Composer blocks on")

	p.run(2*time.Minute, "composer", "config", "audit.ignore", "--json", `["PKSA-w9tt-7782-78jx"]`)
	p.run(10*time.Minute, "composer", "install", "--no-interaction")
	if _, err := os.Stat(filepath.Join(p.runDir, "vendor", "league", "flysystem")); err != nil {
		t.Errorf("flysystem is not installed after its advisory was accepted: %v", err)
	}
}
