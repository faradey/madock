package docker

import "testing"

func withProxyRunning(t *testing.T, running bool) {
	t.Helper()
	prev := proxyRunning
	proxyRunning = func() bool { return running }
	t.Cleanup(func() { proxyRunning = prev })
}

// start and rebuild reported success over a proxy that had not come up, and
// the project's hosts did not answer until somebody ran proxy:start.
func TestAStoppedProxyIsReportedForAProjectBehindIt(t *testing.T) {
	withProxyRunning(t, false)

	if !ReportProxyDown(map[string]string{"proxy/enabled": "true"}) {
		t.Error("a project behind a stopped proxy was reported as reachable")
	}
}

func TestARunningProxyIsNotReported(t *testing.T) {
	withProxyRunning(t, true)

	if ReportProxyDown(map[string]string{"proxy/enabled": "true"}) {
		t.Error("a running proxy was reported as down")
	}
}

// A project with the proxy off is reached by its ports; the shared proxy being
// down is nothing to it.
func TestAProjectWithoutTheProxyIgnoresIt(t *testing.T) {
	withProxyRunning(t, false)

	if ReportProxyDown(map[string]string{"proxy/enabled": "false"}) {
		t.Error("a project with proxy/enabled false was told about the proxy")
	}
}
