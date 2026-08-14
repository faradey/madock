//go:build e2e

package e2e

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
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

// reportWhyTheSiteIsSilent prints what the project looks like from inside when
// a request through the proxy gets no answer.
//
// "EOF" is what Go says when the connection closed before a response, and every
// candidate for that produces the same word: the project's nginx not running,
// php-fpm not answering it, a certificate the proxy would not serve, or the
// proxy pointing at nothing. Each is visible from a different place, so all four
// are asked at once — a failure that only happens on somebody else's machine is
// worth more evidence than a failure you can reproduce.
//
// Everything here is best effort and none of it fails the test; the assertion
// that called it has already decided.
func reportWhyTheSiteIsSilent(t *testing.T, p *project, host string) {
	t.Helper()

	if out, err := p.tryRun(2*time.Minute, "status"); err == nil {
		t.Logf("containers:\n%s", out)
	} else {
		t.Logf("status could not be read: %v\n%s", err, out)
	}

	if digest := servedCertificateDigestOrEmpty(host); digest == "" {
		t.Logf("the proxy served no certificate for %s — the TLS handshake is where this ends", host)
	} else {
		t.Logf("the proxy served a certificate for %s (%s), so the handshake is fine and the answer is missing behind it", host, digest[:12])
	}

	// The proxy decides whether a request ever reaches the project, and it does
	// it by server name. A host the generated configuration does not mention
	// falls to the default block, which closes the connection without answering
	// — the client calls that EOF, and so far that is the whole story it tells.
	//
	// Two different faults produce it, and this separates them: a host missing
	// from the file is a generation problem, a host present in a file the
	// running container is not using is a reload problem.
	proxyConf := filepath.Join(p.execDir, "aruntime", "ctx", "proxy.conf")
	if data, err := os.ReadFile(proxyConf); err != nil {
		t.Logf("the generated proxy configuration could not be read (%s): %v", proxyConf, err)
	} else if !strings.Contains(string(data), host) {
		t.Logf("the generated proxy configuration does not mention %s at all — the server names it carries are:\n%s",
			host, serverNamesIn(string(data)))
	} else {
		t.Logf("the generated proxy configuration does carry %s, so the running proxy is not using this file", host)
	}

	for _, service := range []string{"nginx", "php"} {
		if out, err := p.tryRun(2*time.Minute, "logs", "-s", service); err == nil {
			t.Logf("last of the %s log:\n%s", service, lastLines(out, 25))
		} else {
			t.Logf("the %s log could not be read: %v", service, err)
		}
	}
}

// servedCertificateDigestOrEmpty is servedCertificateDigest without the fatal:
// here the absence of a certificate is a piece of evidence, not a failure.
func servedCertificateDigestOrEmpty(host string) string {
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	conn, err := dialer.Dial("tcp", "127.0.0.1:443")
	if err != nil {
		return ""
	}
	tlsConn := tls.Client(conn, &tls.Config{ServerName: host, InsecureSkipVerify: true})
	defer func() { _ = tlsConn.Close() }()
	if err := tlsConn.Handshake(); err != nil {
		return ""
	}
	certs := tlsConn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return ""
	}
	sum := sha256.Sum256(certs[0].Raw)
	return hex.EncodeToString(sum[:])
}

// serverNamesIn lists the hosts a generated proxy configuration answers for.
func serverNamesIn(conf string) string {
	var names []string
	for _, line := range strings.Split(conf, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "server_name") {
			names = append(names, strings.TrimSuffix(strings.TrimPrefix(line, "server_name"), ";"))
		}
	}
	if len(names) == 0 {
		return "  (none)"
	}
	return "  " + strings.Join(names, "\n  ")
}

func lastLines(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}

func firstLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n")
}
