//go:build e2e

package e2e

import (
	"net"
	"strings"
	"testing"
	"time"
)

// TestProxyCommandsControlTheProxy runs the four proxy verbs nothing has ever
// run.
//
// The proxy has two tests already and neither names a proxy command: they start
// projects and check what comes back over HTTPS, so the proxy is exercised
// sideways, as a side effect of `start`. That leaves the commands a person
// reaches for when the proxy is the thing that is wrong — `proxy:stop`,
// `proxy:start`, `proxy:reload`, `proxy:rebuild` — with no coverage at all. One
// of them is how HTTPS comes back after a mistake, and until now none of them
// was known to do anything.
//
// All four run against one project, in one test, on purpose: each `start` costs
// minutes and the sequence is the point — stopping proves the assertion can go
// red, so every "the certificate is served" after it means something.
//
// The assertion is the TLS handshake, for the reason given in proxy_test.go:
// what sits behind the proxy serves nothing, so HTTP status says nothing about
// routing while the certificate is chosen by server name.
func TestProxyCommandsControlTheProxy(t *testing.T) {
	p := newProject(t, "e2eproxyctl")

	p.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2eproxyctl.test",
	)
	p.run(20*time.Minute, "start")
	requireCertificateFor(t, p, "e2eproxyctl.test")

	// stop. Without this the rest of the test cannot fail: a proxy that was
	// never taken down answers every later check whether or not the command
	// that was supposed to bring it back did anything.
	p.run(3*time.Minute, "proxy:stop")
	requireProxyUnreachable(t, "after proxy:stop")

	// reload with nothing to reload. `nginx -s reload` runs inside a container
	// that is not there, so the exec fails — and the command has to say so
	// rather than print its success line regardless. Six of the defects this
	// suite found were this exact shape.
	out, err := p.tryRun(3*time.Minute, "proxy:reload")
	if err == nil && strings.Contains(out, "Done") {
		t.Errorf("proxy:reload reported success with the proxy stopped:\n%s", out)
	}
	requireProxyUnreachable(t, "after a reload that could not run")

	// start. The proxy container exists and is stopped, which is a different
	// path from creating it: UpNginx decides between the two by asking compose
	// whether the container is running, and a stopped container must not count.
	p.run(5*time.Minute, "proxy:start")
	requireCertificateFor(t, p, "e2eproxyctl.test")

	// reload with the proxy up. It re-parses routing and certificates in place,
	// so the project has to still be served afterwards — a reload that takes
	// the site down is worse than one that does nothing.
	p.run(3*time.Minute, "proxy:reload")
	requireCertificateFor(t, p, "e2eproxyctl.test")

	// restart is stop plus start in one command, and it is what the certificate
	// test uses. Kept here for the sequence, not for the certificate.
	p.run(5*time.Minute, "proxy:restart")
	requireCertificateFor(t, p, "e2eproxyctl.test")

	// rebuild throws the container away and builds it again. It is the heaviest
	// of the four and the one most likely to come back without the routing it
	// had: the configuration is regenerated from the project registry, not from
	// whatever the old container was serving.
	p.run(10*time.Minute, "proxy:rebuild")
	requireCertificateFor(t, p, "e2eproxyctl.test")
}

// requireProxyUnreachable waits for port 443 to stop accepting connections.
//
// Docker unpublishes the port when the container stops, so the dial is refused
// — there is no handshake to inspect and no need for one. It polls because
// stopping is not instant, and a single immediate dial would pass against a
// proxy that is still on its way down.
func requireProxyUnreachable(t *testing.T, when string) {
	t.Helper()

	dialer := &net.Dialer{Timeout: 5 * time.Second}
	deadline := time.Now().Add(30 * time.Second)
	for {
		conn, err := dialer.Dial("tcp", "127.0.0.1:443")
		if err != nil {
			return
		}
		_ = conn.Close()

		if time.Now().After(deadline) {
			t.Errorf("%s: the proxy is still accepting connections on 443", when)
			return
		}
		time.Sleep(2 * time.Second)
	}
}
