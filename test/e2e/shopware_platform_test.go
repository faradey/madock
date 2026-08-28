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

	// Skipped from 2026-08-14 to 2026-08-29, while no Shopware release installed at
	// all. Kept because the version below was chosen against it.
	//
	// Shopware declares its dependencies as exact versions, and packagist keeps
	// publishing advisories against versions already released. Composer 2.8 and
	// later refuse to install a package that has one — the container's composer is
	// 2.10.2 — and an exact requirement leaves it nothing else to choose. What
	// stopped each release, resolved with that composer rather than read off
	// metadata:
	//
	//   6.7.13.0            mcp/sdk ^0.6.0, and 0.6.0 is the only match: advisory
	//                       PKSA-p9gd-j6gr-6f9t, published 2026-08-14 06:19 UTC
	//   6.7.12.2, .1, 11.1  symfony/mcp-bundle ~0.9.0, which wants mcp/sdk ^0.5 —
	//                       the same advisory by another path
	//   6.7.11.0            dompdf/dompdf pinned at 3.1.4, which carries six
	//
	// Two blockers there, not three: the first two are one advisory reached by two
	// paths. Counting three independent walls makes a pattern out of two points, and
	// predicts a fourth blocked package that never appeared.
	//
	// 6.7.13.1 lifts the constraint, mcp/sdk resolves to 0.7.1, and the tree
	// installs with blocking still on — measured on 2026-08-29 against a real stand,
	// both sides of it: the old pin refusing with the text above, the new one at
	// exit 0. So the pin moved rather than the skip being traded for
	// `--no-security-blocking`. That flag was available the whole time and was
	// always the wrong answer: it installs packages with known advisories onto every
	// stand this test stands in for.
	//
	// If this goes red at the install step, suspect packagist before suspecting the
	// change under test. The pin is exact, advisories are published against releases
	// that already exist, and 6.7.13.1 can be blocked tomorrow the way 6.7.13.0 was
	// — by an upstream package, with nothing here having moved. What the failure
	// looks like: composer naming a package and an advisory ID. The answer is the
	// next Shopware release, and the check is one command in a container, never on
	// the host, whose composer is too old to know advisories exist:
	//
	//   madock cli --service php composer update --dry-run

	p := newProject(t, "e2eshopware")

	p.run(45*time.Minute, "setup", "-y", "-d", "-i",
		"--platform=shopware",
		"--platform-version=6.7.13.1",
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
