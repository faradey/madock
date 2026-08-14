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

	p := newProject(t, "e2eshopware")

	// 6.7.12.2 rather than the newest release, and the reason is upstream rather
	// than here. shopware/core v6.7.13.0 is the only one of its 209 releases that
	// requires mcp/sdk, as `^0.6.0` — which for a 0.x version means 0.6.0 and
	// nothing else. On 2026-08-14 at 06:19 UTC, advisory PKSA-p9gd-j6gr-6f9t
	// (CVE-2026-53965, an unbounded SSE buffer in that package's client) was
	// published against >=0.5.0,<0.7.1, so composer refuses to install the only
	// version the constraint allows and `create-project` dies with exit status 2
	// after three hundred lines of version candidates. This test was green sixteen
	// hours earlier on the same code.
	//
	// Not worked around by relaxing composer's advisory policy: that would install
	// a package with a CVE onto every stand this test stands in for. 6.7.12.2 does
	// not require mcp/sdk at all.
	//
	// Move this back to the newest release once shopware/core constrains mcp/sdk to
	// something that includes 0.7.1.
	p.run(45*time.Minute, "setup", "-y", "-d", "-i",
		"--platform=shopware",
		"--platform-version=6.7.12.2",
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
