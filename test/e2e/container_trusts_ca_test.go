//go:build e2e

package e2e

import (
	"strings"
	"testing"
	"time"
)

// TestCodeInTheContainerTrustsMadocksCA — madock installed its CA only into
// the host's store. PHP in the container could not verify https://<project>.test:
// file_get_contents failed, and only SSL_CERT_FILE or verify_peer=false got
// through (found on OrbStack, where the host's roots reach the bundle but no
// hash file does; on Docker Desktop not even the bundle has it). The CA now
// goes into the image, and PHP's own default lookup finds it.
func TestCodeInTheContainerTrustsMadocksCA(t *testing.T) {
	p := newProject(t, "e2etrustca")
	p.run(setupTimeout, "setup", "-y", "--platform=custom", "--language=php", "--hosts=e2etrustca.test")
	p.run(20*time.Minute, "start")

	out, err := p.tryRun(2*time.Minute, "cli", "php", "-r",
		// ignore_errors: any HTTP answer means the TLS handshake verified — an
		// empty project answers 404, which is not what this is about.
		`$c = stream_context_create(["http" => ["ignore_errors" => true]]); $r = @file_get_contents("https://e2etrustca.test/", false, $c); echo ($r === false ? "FAILED: " . (error_get_last()["message"] ?? "") : "OK"), "\n";`)
	if err != nil || !strings.Contains(out, "OK") {
		t.Errorf("PHP in the container does not trust madock's CA (%v):\n%s", err, out)
	}
	if strings.Contains(out, "SSL_CERT_FILE") {
		t.Error("the check must use PHP's default lookup, not SSL_CERT_FILE")
	}
}
