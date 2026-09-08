//go:build e2e

package e2e

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestProxyStartsWithTheWorkerSettingsItIsGiven is the scenario the settings
// were extracted for, and the one the unit tests cannot reach.
//
// Those compare text: they say the generated file carries the numbers. Whether
// nginx will *load* that file is a different question, and on the shared proxy
// it is the expensive one — the proxy is one container per machine, so a file it
// refuses takes every project down together. `server_names_hash_bucket_size` is
// exactly such a value: nginx stops with "could not build server_names_hash, you
// should increase server_names_hash_bucket_size" when a hostname does not fit.
//
// So the assertion is a running proxy and a site that answers, with the numbers
// changed from their defaults.
func TestProxyStartsWithTheWorkerSettingsItIsGiven(t *testing.T) {
	install := newInstallation(t)
	p := install.project("e2eworkers")

	p.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2eworkers.test",
	)

	// Away from the defaults in both directions: fewer workers, a bigger hash
	// table. A test that only raised numbers would pass against a binary that
	// ignored the setting and rendered the defaults, because the defaults are
	// large enough for one short hostname.
	// --global, because the shared proxy is one per machine and reads the
	// installation's own config. Without the flag the value lands in the
	// project's file, where nothing reads it: the first run of this test set the
	// three keys, watched config:set accept them, and found the generated proxy
	// still carrying the defaults.
	for key, value := range map[string]string{
		"proxy/worker/processes":              "3",
		"proxy/worker/connections":            "2048",
		"proxy/server_names_hash/bucket_size": "256",
	} {
		p.run(2*time.Minute, "config:set", "--global", "-n", key, "-v", value)
	}

	p.run(20*time.Minute, "start")

	// The file first, so a failure below can be told apart: rendered wrong, or
	// rendered right and refused.
	proxyConf := readFile(t, filepath.Join(install.dir, "aruntime", "ctx", "proxy.conf"))
	for _, want := range []string{
		"worker_processes 3;",
		"worker_connections 2048;",
		"server_names_hash_bucket_size  256;",
	} {
		requireContains(t, proxyConf, want, "the generated proxy configuration")
	}

	// And the proxy itself. `status` reports the container; the request proves
	// nginx accepted the configuration rather than exiting on it.
	requireContains(t, p.run(3*time.Minute, "status"), "nginx", "the proxy after a start with changed worker settings")

	if status, _ := proxyGet(t, "e2eworkers.test"); status == 0 {
		describeProxy(t, p)
		t.Fatal("nothing answered on 443 after the worker settings were changed — the proxy did not accept its configuration")
	}
}

// TestPhpMemoryLimitReachesTheRunningInterpreter asks php, not the file.
//
// The limit is handed over as a fastcgi parameter by the project's vhost, so
// nothing about it is visible from the container's own php.ini and nothing in
// the generated text proves it arrived. The only honest witness is a request
// that goes through nginx into php-fpm and reports what the interpreter has.
//
// It was three copies of one number inside the platform vhosts until 4.2.5, and
// a project could not change them at all without adopting a 240-line file.
func TestPhpMemoryLimitReachesTheRunningInterpreter(t *testing.T) {
	install := newInstallation(t)
	p := install.project("e2ephplimit")

	p.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=php",
		"--hosts=e2ephplimit.test",
	)

	p.run(2*time.Minute, "config:set", "-n", "php/limits/memory", "-v", "333M")

	// The page that answers the question, in the directory the vhost actually
	// serves: `public/`, from `public_dir` in the defaults. The first run put it
	// at the project root and got nginx's own 404 — a reminder that "the file is
	// there" and "the server serves it" are two claims.
	writeProjectFile(t, p, filepath.Join("public", "index.php"),
		"<?php echo 'memory_limit=', ini_get('memory_limit'), \"\\n\";\n")

	p.run(25*time.Minute, "start")

	status, body := proxyGet(t, "e2ephplimit.test")
	if status == 0 {
		describeProxy(t, p)
		t.Fatal("the project did not answer over HTTPS, so nothing can be asked of php")
	}

	requireContains(t, body, "memory_limit=333M", "what the running interpreter reports")
	if strings.Contains(body, "memory_limit=756M") {
		t.Errorf("php still has the old hardcoded limit:\n%s", body)
	}
}

// TestProxyOffersOnlyTheTlsVersionsItIsGiven measures the handshake rather than
// the file, and the difference is not academic here: the headers feature was
// caught by exactly this gap — the file said one thing and the response carried
// another.
func TestProxyOffersOnlyTheTlsVersionsItIsGiven(t *testing.T) {
	install := newInstallation(t)
	p := install.project("e2etls")

	p.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2etls.test",
	)

	p.run(2*time.Minute, "config:set", "--global", "-n", "proxy/ssl/protocols", "-v", "TLSv1.3")
	p.run(20*time.Minute, "start")

	// 1.3 must work.
	if version, err := handshakeVersion(t, "e2etls.test", tls.VersionTLS13, tls.VersionTLS13); err != nil {
		describeProxy(t, p)
		t.Fatalf("TLS 1.3 was configured and refused: %v", err)
	} else if version != tls.VersionTLS13 {
		t.Errorf("negotiated 0x%x, want TLS 1.3", version)
	}

	// And 1.2 must not, which is the half that says the setting was applied
	// rather than merely written. A proxy still on the default offers both.
	if version, err := handshakeVersion(t, "e2etls.test", tls.VersionTLS12, tls.VersionTLS12); err == nil {
		t.Errorf("TLS 1.2 was still accepted (negotiated 0x%x) after the proxy was limited to 1.3", version)
	}
}

// proxyGet fetches one page through the shared proxy on 443, returning 0 when
// nothing answered. Short by design: the waiting these tests need is measured in
// seconds, not the fifteen minutes a real store takes to warm up.
func proxyGet(t *testing.T, host string) (int, string) {
	t.Helper()

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
				return (&net.Dialer{Timeout: 10 * time.Second}).DialContext(ctx, network, "127.0.0.1:443")
			},
			TLSClientConfig: &tls.Config{ServerName: host, InsecureSkipVerify: true},
		},
	}

	deadline := time.Now().Add(3 * time.Minute)
	for attempt := 1; ; attempt++ {
		resp, err := client.Get("https://" + host + "/")
		if err == nil {
			body := make([]byte, 32*1024)
			n, _ := resp.Body.Read(body)
			_ = resp.Body.Close()
			return resp.StatusCode, string(body[:n])
		}
		if time.Now().After(deadline) {
			t.Logf("%s never answered: %v (%d attempts)", host, err, attempt)
			return 0, ""
		}
		time.Sleep(3 * time.Second)
	}
}

// handshakeVersion opens one TLS connection within the given version range and
// reports what was negotiated.
func handshakeVersion(t *testing.T, host string, min, max uint16) (uint16, error) {
	t.Helper()

	conn, err := net.DialTimeout("tcp", "127.0.0.1:443", 15*time.Second)
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	client := tls.Client(conn, &tls.Config{
		ServerName:         host,
		InsecureSkipVerify: true,
		MinVersion:         min,
		MaxVersion:         max,
	})
	if err := client.Handshake(); err != nil {
		return 0, err
	}
	defer client.Close()

	return client.ConnectionState().Version, nil
}

// writeProjectFile puts a file in the project's own directory, which is what the
// containers mount as the document root.
func writeProjectFile(t *testing.T, p *project, name, body string) {
	t.Helper()

	path := filepath.Join(p.runDir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("creating %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("writing %s: %v", name, err)
	}
}
