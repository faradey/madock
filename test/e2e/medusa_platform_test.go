//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
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

	// The storefront is a second clone into a directory that already exists,
	// because it is a bind-mount source. It used to be skipped for exactly that
	// reason — "already exists" — leaving a project with a backend and no shop,
	// which is only noticed when somebody opens the site.
	requireFile(t, filepath.Join(p.runDir, "storefront", "package.json"),
		"the storefront starter, cloned into a directory that was already there")

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

	// Ownership is asked about before removal, not through it. `project:remove`
	// hands the directory back to the user first, so it would succeed even if
	// the dev server were still writing as root — the two fixes have to be told
	// apart, and this is the one that says the entrypoint drops privileges.
	if owned := rootOwnedFiles(t, p.runDir); len(owned) > 0 {
		t.Errorf("the dev server wrote %d file(s) as root, e.g. %v", len(owned), owned[:min(3, len(owned))])
	}

	// Removal is part of what this platform tests, because it is where the
	// ownership problem lands. A dev server writes `.medusa/client/` as root,
	// and `project:remove` — whose whole promise is to leave nothing behind —
	// used to stop there with "permission denied" and leave the directory. The
	// harness cleans up as root afterwards, so without asking here nothing
	// would notice.
	// The proxy goes first, and it has to: it is one container per daemon, it
	// outlives the installation that started it, and `proxy:prune` needs a
	// project to run from — which this is about to delete. Skipping this left
	// the next platform test talking to a proxy that still had Medusa's
	// configuration mounted, so its site answered nothing at all and looked
	// like a defect in that platform.
	p.install.destroyProxy(p)

	if out, err := p.tryRun(10*time.Minute, "project:remove", "--force", "--name="+p.name); err != nil {
		t.Errorf("project:remove could not finish: %v\n%s", err, out)
	}
	if _, err := os.Stat(p.runDir); err == nil {
		t.Errorf("project:remove left %s behind:\n%s", p.runDir, describeTree(t, p.runDir))
	}
}

// rootOwnedFiles returns paths under root that belong to uid 0.
//
// Everything a container writes into the project should come out owned by
// whoever runs madock — the images remap their application user to that uid for
// exactly this reason. A file owned by root means some process inside is still
// running as root, and the user cannot delete their own project.
func rootOwnedFiles(t *testing.T, root string) []string {
	t.Helper()

	var owned []string
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		if stat, ok := info.Sys().(*syscall.Stat_t); ok && stat.Uid == 0 {
			owned = append(owned, strings.TrimPrefix(path, root+"/"))
		}
		return nil
	})
	return owned
}
