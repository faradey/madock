//go:build e2e

package e2e

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestSslRebuildIssuesANewCertificate runs the one command in the certificate
// path that no test has ever called.
//
// Certificates are reached two ways today, both indirect: `start` issues one
// when the host set changed, and `restart` re-issues after a rename. `ssl:rebuild`
// is the direct route — the thing to run when a certificate is broken, expired
// or was generated for the wrong names — and it is the only one that forces the
// CA itself to be replaced.
//
// The test pins both halves, because they fail separately: the files on disk
// have to be new, and the proxy has to end up serving the new ones. nginx reads
// certificates when it loads its configuration, so a rebuild that regenerates
// the files does not by itself change what any browser is shown.
func TestSslRebuildIssuesANewCertificate(t *testing.T) {
	p := newProject(t, "e2essl")

	p.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2essl.test",
	)
	p.run(20*time.Minute, "start")

	ctx := filepath.Join(p.execDir, "aruntime", "ctx")
	caBefore := fileDigest(t, filepath.Join(ctx, "madockCA.pem"))
	servedBefore := servedCertificateDigest(t, p, "e2essl.test")
	if servedBefore == "" {
		t.Fatal("the proxy served no certificate before the rebuild")
	}

	p.run(5*time.Minute, "ssl:rebuild")

	if caAfter := fileDigest(t, filepath.Join(ctx, "madockCA.pem")); caAfter == caBefore {
		t.Errorf("ssl:rebuild left the CA certificate untouched (%s)", caBefore)
	}

	// What the proxy serves immediately afterwards is recorded rather than
	// asserted: nginx holds the certificate it loaded, so whether a rebuild
	// alone is enough depends on madock reloading it, and that is a decision
	// this test should report rather than encode.
	if immediate := servedCertificateDigest(t, p, "e2essl.test"); immediate == servedBefore {
		t.Logf("ssl:rebuild replaced the files but the running proxy still serves the old certificate; a reload is required")
	}

	p.run(3*time.Minute, "proxy:reload")

	// Polled rather than sampled once: a reload is graceful, so for a moment
	// the old workers are still answering with the certificate they loaded.
	// Reading once immediately after the command measures that moment and calls
	// it a defect.
	if !certificateChangedWithin(t, p, "e2essl.test", servedBefore, time.Minute) {
		t.Errorf("a minute after ssl:rebuild and proxy:reload the proxy still serves the old certificate (%s)", servedBefore)
	}

	// And it still has to be a certificate for this project. A rebuild that
	// produced something valid for nothing would satisfy every check above.
	requireCertificateFor(t, p, "e2essl.test")
}

// TestConfigCacheCleanRemovesTheCacheAndNothingElse covers `config:cache:clean`,
// which deletes a directory and says nothing about what was in it.
//
// The cache holds the markers madock uses to decide whether the proxy image has
// been pulled and whether the generated proxy configuration is the one already
// applied. Clearing it is advice given whenever something is stuck, so two
// things matter: it must actually empty the directory, and the project must
// still work afterwards — everything in there is derived, and a project that
// cannot start after a cache clean would make the advice dangerous.
func TestConfigCacheCleanRemovesTheCacheAndNothingElse(t *testing.T) {
	p := newProject(t, "e2econfcache")

	p.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2econfcache.test",
	)
	p.run(20*time.Minute, "start")

	cache := filepath.Join(p.execDir, "cache")
	if len(dirEntries(t, cache)) == 0 {
		t.Fatalf("nothing in %s after a start, so this test would prove nothing", cache)
	}

	p.run(2*time.Minute, "config:cache:clean")

	if left := dirEntries(t, cache); len(left) != 0 {
		t.Errorf("config:cache:clean left %v behind in %s", left, cache)
	}
	// The directory itself has to survive: the code recreates it, and the next
	// command writing a marker into a directory that is not there fails.
	if _, err := os.Stat(cache); err != nil {
		t.Errorf("config:cache:clean removed the cache directory itself: %v", err)
	}

	// The project's own configuration is not cache and must not have gone with
	// it.
	requireFile(t, filepath.Join(p.execDir, "projects", p.name, "config.xml"),
		"the project config after a cache clean")

	// And the environment still comes back. The markers that were deleted are
	// exactly the ones the proxy path reads.
	p.run(10*time.Minute, "restart")
	requireCertificateFor(t, p, "e2econfcache.test")
	requireContains(t, p.run(3*time.Minute, "status"), "db running", "the database after a cache clean and restart")
}

// fileDigest returns the sha256 of a file, and fails the test if it is missing:
// every caller here is asking "did this change", and an absent file is an
// answer nobody wants silently folded into "yes".
func fileDigest(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// servedCertificateDigest returns the sha256 of the certificate the proxy
// offers for a host, or "" if it offers none.
func servedCertificateDigest(t *testing.T, p *project, host string) string {
	t.Helper()

	dialer := &net.Dialer{Timeout: 10 * time.Second}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	deadline := time.Now().Add(30 * time.Second)
	for {
		conn, err := dialer.DialContext(ctx, "tcp", "127.0.0.1:443")
		if err == nil {
			tlsConn := tls.Client(conn, &tls.Config{ServerName: host, InsecureSkipVerify: true})
			if err = tlsConn.HandshakeContext(ctx); err == nil {
				certs := tlsConn.ConnectionState().PeerCertificates
				_ = tlsConn.Close()
				if len(certs) == 0 {
					return ""
				}
				sum := sha256.Sum256(certs[0].Raw)
				return hex.EncodeToString(sum[:])
			}
			_ = conn.Close()
		}
		if time.Now().After(deadline) {
			describeProxy(t, p)
			t.Fatalf("%s: could not complete a TLS handshake with the proxy: %v", host, err)
		}
		time.Sleep(2 * time.Second)
	}
}

// certificateChangedWithin waits for the proxy to start offering a certificate
// other than the one given.
func certificateChangedWithin(t *testing.T, p *project, host, previous string, within time.Duration) bool {
	t.Helper()

	deadline := time.Now().Add(within)
	for {
		if current := servedCertificateDigest(t, p, host); current != "" && current != previous {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(2 * time.Second)
	}
}

func dirEntries(t *testing.T, path string) []string {
	t.Helper()

	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatalf("reading %s: %v", path, err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}
