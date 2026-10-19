//go:build e2e

package e2e

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestAProjectWithoutPublishedPortsIsReachedOnlyThroughTheProxy — with
// ports/publish false nothing of the project listens on the host, and the
// shared proxy reaches its nginx over madock-proxy by the container's name.
//
// The part most likely to break is not the routing but the proxy's own start.
// nginx resolves the names in an upstream block when it loads its
// configuration, and a stopped project keeps its block — so without `resolve`
// stopping such a project would take the shared proxy down at its next reload,
// for every project behind it. The second project is here to cause that reload.
func TestAProjectWithoutPublishedPortsIsReachedOnlyThroughTheProxy(t *testing.T) {
	install := newInstallation(t)
	hidden := install.project("e2epub")
	other := install.project("e2epubother")

	hidden.run(5*time.Minute, "setup", "-y", "--platform=custom", "--language=none", "--hosts=e2epub.test")
	hidden.run(2*time.Minute, "config:set", "-n", "ports/publish", "-v", "false")
	hidden.run(20*time.Minute, "start")

	compose, err := os.ReadFile(hidden.generated("docker-compose.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(compose), "ports:") {
		t.Errorf("the generated compose still publishes a port:\n%s", compose)
	}

	var reported struct {
		Data struct {
			Ports     []any `json:"ports"`
			Published bool  `json:"published"`
		} `json:"data"`
	}
	out := hidden.run(2*time.Minute, "info:ports", "--json")
	if err := json.Unmarshal([]byte(jsonPart(out)), &reported); err != nil {
		t.Fatalf("info:ports --json: %v\n%s", err, out)
	}
	if reported.Data.Published || len(reported.Data.Ports) != 0 {
		t.Errorf("info:ports reports ports nothing listens on:\n%s", out)
	}

	// Through the proxy. The bare custom stack answers with whatever it
	// answers; what proves the route is the request arriving in the project's
	// own nginx log.
	requireCertificateFor(t, hidden, "e2epub.test")
	// Repeated, not sent once: `resolve` looks the name up asynchronously, so
	// the first request after the proxy reloads can arrive before the answer.
	deadline := time.Now().Add(60 * time.Second)
	for {
		status := requestThroughProxy(t, "e2epub.test", "/n6-probe")
		logs := hidden.run(2*time.Minute, "logs", "-s", "nginx")
		if strings.Contains(logs, "/n6-probe") {
			break
		}
		if time.Now().After(deadline) {
			proxyConf, _ := os.ReadFile(filepath.Join(hidden.execDir, "aruntime", "ctx", "proxy.conf"))
			var upstreams []string
			for _, line := range strings.Split(string(proxyConf), "\n") {
				if strings.Contains(line, "server ") && (strings.Contains(line, "resolve") || strings.Contains(line, "e2epub")) || strings.HasPrefix(strings.TrimSpace(line), "resolver") {
					upstreams = append(upstreams, line)
				}
			}
			t.Fatalf("the request sent through the proxy (answered %d) never reached the project's nginx.\nproxy upstream lines:\n%s\nproject nginx log:\n%s",
				status, strings.Join(upstreams, "\n"), logs)
		}
		time.Sleep(2 * time.Second)
	}

	// Stopped, its block stays in the proxy; starting another project reloads
	// the proxy with that block in it.
	hidden.run(5*time.Minute, "stop")
	other.run(5*time.Minute, "setup", "-y", "--platform=custom", "--language=none", "--hosts=e2epubother.test")
	other.run(20*time.Minute, "start")
	requireCertificateFor(t, other, "e2epubother.test")
}

// requestThroughProxy sends one HTTPS request to the proxy on this machine for
// host and returns the status. The answer is not checked — only that the proxy
// took it.
func requestThroughProxy(t *testing.T, host, path string) int {
	t.Helper()
	client := &http.Client{
		Timeout: 20 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
				return (&net.Dialer{Timeout: 10 * time.Second}).DialContext(ctx, network, "127.0.0.1:443")
			},
			TLSClientConfig: &tls.Config{ServerName: host, InsecureSkipVerify: true},
		},
	}
	resp, err := client.Get("https://" + host + path)
	if err != nil {
		t.Fatalf("request through the proxy: %v", err)
	}
	_ = resp.Body.Close()
	return resp.StatusCode
}
