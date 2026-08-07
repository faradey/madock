//go:build e2e

package e2e

import (
	"context"
	"crypto/tls"
	"net"
	"strings"
	"testing"
	"time"
)

// TestProxyRoutesEveryProjectOverHTTPS is the test the proxy has been missing.
//
// The proxy is the one piece of madock that every project shares: a single
// nginx whose configuration is regenerated whenever anything starts. That makes
// it the place where one project can quietly break another, and it has: a
// certificate that survived a restart in one release and was wiped in the next,
// and a routing entry that only appeared after a rebuild.
//
// Two projects in one installation is the smallest arrangement in which that
// class of defect can show up at all.
func TestProxyRoutesEveryProjectOverHTTPS(t *testing.T) {
	install := newInstallation(t)

	first := install.project("e2eproxya")
	second := install.project("e2eproxyb")

	for _, p := range []*project{first, second} {
		p.run(5*time.Minute, "setup", "-y",
			"--platform=custom",
			"--language=none",
			"--hosts="+p.name+".test",
		)
		p.run(20*time.Minute, "start")
	}

	// Starting the second project regenerates the shared configuration. If that
	// regeneration is written from the project being started rather than from
	// every project, the first one disappears from the proxy — which is exactly
	// how it once behaved.
	requireCertificateFor(t, "e2eproxya.test")
	requireCertificateFor(t, "e2eproxyb.test")

	// proxy:restart is the operation that took HTTPS down before. It has no
	// project argument, so nothing about it says which projects it is allowed
	// to forget.
	second.run(5*time.Minute, "proxy:restart")

	requireCertificateFor(t, "e2eproxya.test")
	requireCertificateFor(t, "e2eproxyb.test")
}

// requireCertificateFor opens a TLS connection to the proxy asking for the given
// host and checks the certificate it is offered actually covers that host.
//
// The handshake is the assertion. What sits behind the proxy is a bare `app`
// container serving nothing, so the HTTP response would be a 502 and would say
// nothing about routing — whereas the certificate is chosen by server name, and
// getting the right one back means the proxy knows the project.
func requireCertificateFor(t *testing.T, host string) {
	t.Helper()

	dialer := &net.Dialer{Timeout: 10 * time.Second}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var lastErr error
	deadline := time.Now().Add(30 * time.Second)
	for {
		conn, err := dialer.DialContext(ctx, "tcp", "127.0.0.1:443")
		if err == nil {
			// The certificate is signed by madock's own CA, which this process
			// has no reason to trust. Verification is done by hand below: the
			// question is which names the certificate carries, not who signed
			// it.
			tlsConn := tls.Client(conn, &tls.Config{
				ServerName:         host,
				InsecureSkipVerify: true,
			})
			err = tlsConn.HandshakeContext(ctx)
			if err == nil {
				certs := tlsConn.ConnectionState().PeerCertificates
				_ = tlsConn.Close()
				if len(certs) == 0 {
					t.Errorf("%s: the proxy offered no certificate", host)
					return
				}
				for _, name := range certs[0].DNSNames {
					if name == host {
						return
					}
				}
				t.Errorf("%s: the proxy answered with a certificate for %s instead",
					host, strings.Join(certs[0].DNSNames, ", "))
				return
			}
			_ = conn.Close()
		}
		lastErr = err

		if time.Now().After(deadline) {
			t.Errorf("%s: could not complete a TLS handshake with the proxy: %v", host, lastErr)
			return
		}
		time.Sleep(2 * time.Second)
	}
}
