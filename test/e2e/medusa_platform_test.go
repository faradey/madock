//go:build e2e

package e2e

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestMedusaInstallsAndAnswers is the Node platform, and it is a different
// shape of proof from the two PHP ones.
//
// Magento and Shopware are a php container, a document root and a request that
// nginx hands to php-fpm. Medusa is a Node process that has to *stay running*:
// the install ends with `yarn dev`, and the proxy talks to a long-lived server
// rather than to a pool that is respawned per request. Nothing else in the
// suite covers a platform that works that way — nor the postgres path, since
// every other test here runs MariaDB.
//
// It is also the second platform CI could hold a verdict on: the starter is
// cloned from a public GitHub repository and yarn pulls from the public
// registry, so there are no credentials in the way.
//
// `/health` is the assertion rather than a page: it is the endpoint Medusa
// itself uses to say the server has finished booting, and unlike a storefront
// route it does not depend on seeded regions, a publishable key or built admin
// assets. What it proves is the whole chain — the node container came up, yarn
// installed, the migrations ran against postgres, and the server is answering
// through the proxy.
func TestMedusaInstallsAndAnswers(t *testing.T) {
	if !platformTestsEnabled() {
		t.Skip("platform tests are opt-in: ./test/e2e/e2e.sh run --platforms -run TestMedusa")
	}

	p := newProject(t, "e2emedusa")

	p.run(45*time.Minute, "setup", "-y", "-d", "-i",
		"--platform=medusa",
		"--hosts=e2emedusa.test",
	)

	// Cloned, not built: these come from the starter repository, so their
	// absence separates a download that did not happen from an install that
	// did not finish.
	requireFile(t, filepath.Join(p.runDir, "package.json"),
		"the starter's manifest, which only exists if the clone completed")
	requireFile(t, filepath.Join(p.runDir, "medusa-config.ts"),
		"the Medusa configuration the install patches")
	requireFile(t, filepath.Join(p.runDir, ".env"),
		"the environment file the install writes, with the database URL in it")

	requireContains(t, p.run(3*time.Minute, "status"), "nodejs running",
		"the container the dev server lives in")
	t.Logf("the installed project: %s", describeTree(t, p.runDir))

	status, body := httpsGet(t, "e2emedusa.test", "/health")
	if status != 200 {
		t.Fatalf("/health answered %d, not 200:\n%s", status, firstLines(body, 20))
	}
	// Medusa answers "OK" and nothing else. A 200 carrying an HTML page would
	// mean something other than the backend replied — the proxy's own error
	// page being the obvious candidate.
	if !strings.Contains(strings.ToUpper(body), "OK") {
		t.Errorf("/health answered 200 but not with what Medusa says:\n%s", firstLines(body, 20))
	}
}
