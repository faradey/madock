//go:build e2e

package e2e

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestShopwareInstallsAndAnswers is the platform test CI can actually hold a
// verdict on.
//
// The Magento one cannot: its download needs Adobe credentials, which a public
// repository must not carry, so it only ever runs on a developer's machine.
// Shopware comes from packagist and needs nothing — same shape, same value, and
// a runner can prove it.
//
// It also exercises the fix the Magento test forced, and the arguments below
// are chosen for that rather than for tidiness.
//
// `--platform-version` *with* `--php` is the combination that used to skip the
// version table here: the table was read further down, but only while PHP was
// still empty, and naming a PHP version filled it. What was skipped is Composer,
// the database and the search engine, all left at the values for an unknown
// version — empty strings. Passing the version alone would take the branch that
// always worked and prove nothing.
//
// Nothing else is named on purpose: those three have to come from the table.
func TestShopwareInstallsAndAnswers(t *testing.T) {
	if !platformTestsEnabled() {
		t.Skip("platform tests are opt-in: ./test/e2e/e2e.sh run --platforms -run TestShopware")
	}

	// Skipped from 2026-08-14: no Shopware release installs at all, and the reason
	// is neither ours nor a version we can pick around.
	//
	// Shopware declares its dependencies as exact versions, and packagist keeps
	// publishing advisories against versions already released. Composer 2.8 and
	// later refuse to install a package that has one — the container's composer is
	// 2.10.2 — and an exact requirement leaves it nothing else to choose. So every
	// release stops at a different wall, verified by resolving each one with that
	// composer rather than by reading metadata:
	//
	//   6.7.13.0            mcp/sdk ^0.6.0, and 0.6.0 is the only match: advisory
	//                       PKSA-p9gd-j6gr-6f9t, published 2026-08-14 06:19 UTC
	//   6.7.12.2, .1, 11.1  symfony/mcp-bundle ~0.9.0, which wants mcp/sdk ^0.5 —
	//                       the same advisory by another path
	//   6.7.11.0            dompdf/dompdf pinned at 3.1.4, which carries six
	//
	// Pinning an older release was tried and is what proved the shape of this: each
	// one fails on its own blocked package, and the older it is the more advisories
	// its pins have collected. The test was green sixteen hours before the first
	// failure, on the same code.
	//
	// Composer can install it — `--no-security-blocking` resolves 6.7.13.0 to the
	// last package — so the choice was between installing packages with known
	// advisories onto every stand this test stands in for, and not installing. We
	// chose not to install, and to leave the platform job honest about Medusa
	// rather than red about something upstream.
	//
	// Re-enable when a Shopware release resolves with blocking on. That needs one of
	// mcp/sdk 0.7.1 (which is out) to become acceptable to shopware/core's
	// constraint, or the advisories against dompdf's pinned version to be answered
	// by a release Shopware allows.
	t.Skip("no Shopware release currently installs: composer blocks its exact dependency pins over advisories — see the comment above")

	p := newProject(t, "e2eshopware")

	p.run(45*time.Minute, "setup", "-y", "-d", "-i",
		"--platform=shopware",
		"--platform-version=6.7.13.0",
		"--php=8.3",
		"--hosts=e2eshopware.test",
	)

	// The install writes these, so their absence separates "the download
	// worked and the install did not" from "the store does not answer".
	requireFile(t, filepath.Join(p.runDir, "bin", "console"),
		"the Shopware CLI, which only exists if the download completed")
	requireFile(t, filepath.Join(p.runDir, ".env"),
		"the environment file system:setup writes")

	version := p.run(5*time.Minute, "cli", "bin/console", "--version")
	requireContains(t, version, "Shopware", "the version bin/console reports")
	t.Logf("bin/console said: %s", strings.TrimSpace(version))
	t.Logf("the installed project: %s", describeTree(t, p.runDir))

	// The storefront, over HTTPS, through the proxy. Assets are not built here
	// — that is a separate step and not what this proves — but the page is
	// rendered by PHP against the database it was just given, which is.
	status, body := httpsGet(t, "e2eshopware.test", "/")
	if status != 200 {
		// This passes on a laptop and failed on a CI runner with the same EOF
		// sixty times over fifteen minutes, which is a difference in the
		// machine rather than in the code — and nothing in "EOF" says which
		// part gave up. Ask the environment while it is still standing.
		reportWhyTheSiteIsSilent(t, p, "e2eshopware.test")
		t.Fatalf("the storefront answered %d, not 200:\n%s", status, firstLines(body, 20))
	}
	if !strings.Contains(strings.ToLower(body), "shopware") {
		t.Errorf("something answered 200, but the page does not look like Shopware:\n%s", firstLines(body, 20))
	}
}
