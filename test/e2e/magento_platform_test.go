//go:build e2e

package e2e

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestMagentoInstallsAndAnswers is the test the rest of the suite cannot be.
//
// Everything else runs on `custom` + `--language=none`, which is twenty seconds
// and proves madock's own machinery: config, containers, proxy, databases. What
// it cannot see is whether the thing customers actually run still comes up — a
// PHP image that no longer builds, an OpenSearch tag that was pulled, a composer
// constraint that stopped resolving. None of that is madock's code, and all of
// it reaches a customer as "madock is broken".
//
// Three reasons it is not in the default run, and all three are hard:
//
//   - it takes twenty minutes and pulls hundreds of megabytes;
//   - it needs Adobe credentials, because `setup -d` downloads from
//     repo.magento.com, which refuses anonymous requests;
//   - and therefore it can never be green in CI, which has no credentials and
//     must not be given any. A public repository's secrets are the wrong place
//     for the keys to our Magento account, so this test is deliberately one that
//     only a person can run.
//
// It skips rather than fails when either condition is missing, and says which —
// a red test on a machine that was never going to have the keys teaches nobody
// anything.
func TestMagentoInstallsAndAnswers(t *testing.T) {
	if !platformTestsEnabled() {
		t.Skip("platform tests are opt-in: ./test/e2e/e2e.sh run --platforms -run TestMagento")
	}
	requireMagentoCredentials(t)

	p := newProject(t, "e2emagento")

	// Explicit versions rather than a preset: a preset is a moving target, and a
	// test that silently follows it cannot tell "the newest stack broke" from
	// "the preset changed under us". 2.4.8 with the stack its own release notes
	// name.
	p.run(45*time.Minute, "setup", "-y", "-d", "-i",
		"--platform=magento2",
		"--platform-version=2.4.8",
		"--php=8.4",
		"--db=11.4",
		"--search-engine=OpenSearch",
		"--search-engine-version=2.19.0",
		"--hosts=e2emagento.test",
	)

	// The install writes the store's own files, so their absence means the
	// download half succeeded and the install half did not — which is a
	// different failure from a store that does not answer.
	requireFile(t, filepath.Join(p.runDir, "app", "etc", "env.php"),
		"the configuration setup:install writes")
	requireFile(t, filepath.Join(p.runDir, "bin", "magento"),
		"the Magento CLI, which only exists if the download completed")

	// The exec-shaped commands that have no container to run in on any other
	// project in this suite. `magento` resolves the php container, a user and a
	// working directory, and prints the version of the store that was just
	// installed.
	version := p.run(5*time.Minute, "magento", "--version")
	requireContains(t, version, "2.4.8", "the version bin/magento reports")
	t.Logf("bin/magento said: %s", strings.TrimSpace(version))
	t.Logf("the installed project: %s", describeTree(t, p.runDir))

	// And the whole point: a browser gets a page, over HTTPS, through the proxy.
	// A 200 here means the PHP image built, php-fpm is answering, the database
	// has the schema, and OpenSearch came up — none of which any other test in
	// this suite can see.
	status, body := httpsGet(t, "e2emagento.test", "/")
	if status != http.StatusOK {
		t.Fatalf("the storefront answered %d, not 200:\n%s", status, firstLines(body, 20))
	}
	// A Magento 200 that is not Magento would still be a 200 — nginx's own
	// default page is the obvious candidate.
	if !strings.Contains(body, "Magento") && !strings.Contains(body, "mage-") {
		t.Errorf("something answered 200 on the storefront, but it does not look like Magento:\n%s", firstLines(body, 20))
	}
}

// platformTestsEnabled reports whether the caller asked for the tests that
// install a real store. They pull hundreds of megabytes and take minutes each,
// so nothing runs them by accident.
func platformTestsEnabled() bool {
	return os.Getenv("MADOCK_E2E_PLATFORMS") == "yes"
}

// requireMagentoCredentials skips the test unless this machine can download
// Magento at all.
//
// The file is read for the host name only. Its contents are a password, and a
// test that printed one into a log would be a worse problem than the one it was
// diagnosing.
func requireMagentoCredentials(t *testing.T) {
	t.Helper()

	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("cannot tell whether this machine has composer credentials: %v", err)
	}
	auth := filepath.Join(home, ".composer", "auth.json")

	data, err := os.ReadFile(auth)
	if err != nil {
		t.Skipf("no composer credentials in %s — run ./test/e2e/e2e.sh auth to copy your own in", auth)
	}
	if !strings.Contains(string(data), "repo.magento.com") {
		t.Skipf("%s has no entry for repo.magento.com, so Magento cannot be downloaded here", auth)
	}
}

// httpsGet asks the proxy for a page as a browser would: TLS with the project's
// host name, so the proxy routes by SNI.
//
// The certificate is madock's own, signed by a CA this process has no reason to
// trust — verification is skipped here for the same reason the certificate
// tests verify by hand: what is being asked is whether the store answers, not
// who signed the certificate.
func httpsGet(t *testing.T, host, path string) (int, string) {
	t.Helper()

	client := &http.Client{
		Timeout: 2 * time.Minute,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
				return (&net.Dialer{Timeout: 15 * time.Second}).DialContext(ctx, network, "127.0.0.1:443")
			},
			TLSClientConfig: &tls.Config{ServerName: host, InsecureSkipVerify: true},
		},
	}

	// A store that has just been installed is still warming up, and how long
	// that takes depends on the platform: php-fpm compiles on the first
	// request, while a Node platform builds its admin bundle before it listens
	// at all. Fifteen minutes, because the failure this must not produce is "we
	// gave up first".
	//
	// Every attempt is logged. A run that ends in "it never answered" is
	// otherwise indistinguishable from one that answered 502 for fourteen
	// minutes and then died, and those have different causes.
	deadline := time.Now().Add(15 * time.Minute)
	started := time.Now()
	var lastStatus int
	var lastBody string
	for attempt := 1; ; attempt++ {
		resp, err := client.Get("https://" + host + path)
		if err == nil {
			body := make([]byte, 64*1024)
			n, _ := resp.Body.Read(body)
			_ = resp.Body.Close()
			lastStatus, lastBody = resp.StatusCode, string(body[:n])
			if resp.StatusCode == http.StatusOK {
				t.Logf("%s answered 200 after %s", path, time.Since(started).Round(time.Second))
				return lastStatus, lastBody
			}
			t.Logf("attempt %d (%s): %s answered %d", attempt, time.Since(started).Round(time.Second), path, resp.StatusCode)
		} else {
			lastBody = err.Error()
			t.Logf("attempt %d (%s): %v", attempt, time.Since(started).Round(time.Second), err)
		}
		if time.Now().After(deadline) {
			return lastStatus, lastBody
		}
		time.Sleep(15 * time.Second)
	}
}

// describeTree counts what is on disk and how much of it there is.
//
// A store that installed and a directory that did not are both "present" to a
// file check; the difference is tens of thousands of files. Logged rather than
// asserted, because the number is evidence for a person reading the run, not a
// threshold worth failing on.
func describeTree(t *testing.T, root string) string {
	t.Helper()

	var files int
	var bytes int64
	_ = filepath.Walk(root, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			files++
			bytes += info.Size()
		}
		return nil
	})
	return fmt.Sprintf("%d files, %d MB", files, bytes/(1024*1024))
}

func firstLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n")
}
