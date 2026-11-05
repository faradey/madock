//go:build e2e

package e2e

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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

	hidden.run(setupTimeout, "setup", "-y", "--platform=custom", "--language=none", "--hosts=e2epub.test")
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
	// Repeated, not sent once: the first request after a start can still meet a
	// failed lookup, which the resolver keeps for its `valid`.
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
				if strings.Contains(line, "e2epub-") || strings.HasPrefix(strings.TrimSpace(line), "resolver") {
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
	other.run(setupTimeout, "setup", "-y", "--platform=custom", "--language=none", "--hosts=e2epubother.test")
	other.run(20*time.Minute, "start")
	requireCertificateFor(t, other, "e2epubother.test")
}

// TestAHiddenProjectNeverReceivesAnotherProjectsRequests — docker hands a freed
// address to the next container that starts, so a proxy that remembers an address
// sends a stopped project's visitors to whichever project took it, cookies and
// all. Measured with `resolve` in an upstream block on 2026-11-05: 240 requests of
// 240 landed in the other project, with or without a proxy reload.
//
// A third project with published ports keeps the proxy up: when the last project
// stops the proxy stops with it, and a restarted proxy remembers nothing.
//
// The same runs hold the wait that came with that form, a failed lookup retried
// after 30 s: a project is reached as soon as its start returns, both when it is
// new and when its restart takes the proxy down with it.
func TestAHiddenProjectNeverReceivesAnotherProjectsRequests(t *testing.T) {
	install := newInstallation(t)
	keeper := install.project("e2ekeeper")
	hidden := install.project("e2ehidden")
	taker := install.project("e2etaker")

	keeper.run(setupTimeout, "setup", "-y", "--platform=custom", "--language=none", "--hosts=e2ekeeper.test")
	keeper.run(20*time.Minute, "start")
	for _, p := range []*project{hidden, taker} {
		p.run(setupTimeout, "setup", "-y", "--platform=custom", "--language=none", "--hosts="+p.name+".test")
		p.run(2*time.Minute, "config:set", "-n", "ports/publish", "-v", "false")
	}

	// New: the proxy was reloaded with this project's block before its container
	// existed, so its first lookup failed.
	hidden.run(20*time.Minute, "start")
	requireReachedSoon(t, hidden, "e2ehidden.test", "a new project")

	// Started second, so it holds the higher address and the lower one is what
	// comes free.
	taker.run(20*time.Minute, "start")
	requireReachedSoon(t, taker, "e2etaker.test", "a second new project")
	freed := proxyNetworkAddress(t, hidden)

	hidden.run(5*time.Minute, "stop")
	taker.run(5*time.Minute, "stop")
	// Longer than any `valid` the resolver has had, so that an address still
	// remembered here is remembered by the proxy and not cached by the resolver.
	time.Sleep(12 * time.Second)
	taker.run(5*time.Minute, "start")
	if got := proxyNetworkAddress(t, taker); got != freed {
		t.Fatalf("the test needs the second project on the address the first one freed: %s, got %s", freed, got)
	}

	const path = "/e2e-not-yours"
	for i := 0; i < 20; i++ {
		requestThroughProxy(t, "e2ehidden.test", path)
		time.Sleep(250 * time.Millisecond)
	}
	// A bare custom stack logs each such request twice more, as the [error] of the
	// upstream it could not reach. The access log has it once.
	landed := 0
	for _, line := range strings.Split(taker.run(2*time.Minute, "logs", "-s", "nginx"), "\n") {
		if strings.Contains(line, path) && !strings.Contains(line, "[error]") {
			landed++
		}
	}
	if landed != 0 {
		t.Errorf("%d of 20 requests for a stopped project's host reached the project that took its address", landed)
	}

	// The only project running: its restart takes the proxy down with it, and the
	// proxy comes back knowing no address.
	taker.run(5*time.Minute, "stop")
	keeper.run(5*time.Minute, "stop")
	hidden.run(5*time.Minute, "start")
	hidden.run(5*time.Minute, "restart")
	requireReachedSoon(t, hidden, "e2ehidden.test", "the only project after a restart")
}

// requireReachedSoon fails unless a request through the proxy arrives in the
// project's own nginx within ten seconds. The status proves nothing: a bare custom
// stack answers 502 from that nginx. A retried lookup took 25 s and more from here.
func requireReachedSoon(t *testing.T, p *project, host, what string) {
	t.Helper()

	path := "/e2e-reached-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	deadline := time.Now().Add(10 * time.Second)
	for {
		requestThroughProxy(t, host, path)
		if strings.Contains(p.run(2*time.Minute, "logs", "-s", "nginx"), path) {
			return
		}
		if time.Now().After(deadline) {
			describeProxy(t, p)
			t.Fatalf("%s was not reached within 10 s of its start returning", what)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

// proxyNetworkAddress is the address of the project's nginx on madock-proxy.
func proxyNetworkAddress(t *testing.T, p *project) string {
	t.Helper()

	container := "madock_" + p.name + "-nginx-1"
	out, err := exec.Command("docker", "inspect", "--format", `{{(index .NetworkSettings.Networks "madock-proxy").IPAddress}}`, container).CombinedOutput()
	if err != nil {
		t.Fatalf("could not read the address of %s: %v\n%s", container, err, out)
	}
	address := strings.TrimSpace(string(out))
	if address == "" {
		t.Fatalf("%s has no address on madock-proxy", container)
	}
	return address
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
