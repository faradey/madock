package docker

import (
	"errors"
	"slices"
	"testing"
)

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

func recordProxyUps(t *testing.T, errs ...error) *[][]string {
	t.Helper()
	calls := [][]string{}
	prev := runProxyCompose
	runProxyCompose = func(args ...string) error {
		calls = append(calls, args)
		if len(calls) <= len(errs) {
			return errs[len(calls)-1]
		}
		return nil
	}
	t.Cleanup(func() { runProxyCompose = prev })
	return &calls
}

var proxyUp = []string{"compose", "-f", "proxy.yml", "up", "--no-deps", "-d"}

// A stopped proxy keeps the id of a project network that a recreated stack
// replaced, and `up` of that container fails until it is recreated.
func TestAProxyOnARemovedNetworkIsRecreated(t *testing.T) {
	calls := recordProxyUps(t, errors.New("exit status 1: Error response from daemon: failed to set up container networking: network 9b4a32b7e862 not found"))

	if err := upProxy(proxyUp); err != nil {
		t.Fatalf("upProxy: %v", err)
	}
	if len(*calls) != 2 || !slices.Contains((*calls)[1], "--force-recreate") {
		t.Errorf("calls %v — want a second up with --force-recreate", *calls)
	}
}

func TestAnotherProxyFailureIsNotRetried(t *testing.T) {
	calls := recordProxyUps(t, errors.New("exit status 1: port is already allocated"))

	if err := upProxy(proxyUp); err == nil {
		t.Fatal("the failure was swallowed")
	}
	if len(*calls) != 1 {
		t.Errorf("calls %v — only a removed network is worth a recreate", *calls)
	}
}

func TestARecreateThatFailsIsNotRepeated(t *testing.T) {
	stale := errors.New("exit status 1: network 9b4a32b7e862 not found")
	calls := recordProxyUps(t, stale, stale)

	if err := upProxy(append(proxyUp, "--build", "--force-recreate")); err == nil {
		t.Fatal("the failure was swallowed")
	}
	if len(*calls) != 1 {
		t.Errorf("calls %v — an up that already recreates has nothing left to try", *calls)
	}
}
