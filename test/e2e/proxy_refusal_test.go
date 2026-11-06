//go:build e2e

package e2e

import (
	"strings"
	"testing"
	"time"
)

// TestAConfigurationTheProxyRefusesIsReported gives the shared proxy a setting
// nginx cannot parse while another project keeps it up. A setting of the proxy's
// own, because a bad host would also stop the project's nginx, and start would
// fail for that. The proxy goes on serving, and every command that wanted the new
// configuration says it did not get it.
func TestAConfigurationTheProxyRefusesIsReported(t *testing.T) {
	install := newInstallation(t)
	other := install.project("e2erefother")
	p := install.project("e2eref")
	for _, q := range []*project{other, p} {
		q.run(setupTimeout, "setup", "-y", "--platform=custom", "--language=none", "--hosts="+q.name+".test")
		q.run(20*time.Minute, "start")
	}

	p.run(time.Minute, "config:set", "--global", "-n", "proxy/worker/processes", "-v", "notanumber")

	out, err := p.tryRun(5*time.Minute, "start")
	if err == nil {
		t.Errorf("start exited 0 over a refused configuration:\n%s", out)
	}
	requireContains(t, out, "did not take the new configuration", "start")

	out, err = p.tryRun(time.Minute, "proxy:reload")
	if err == nil {
		t.Errorf("proxy:reload exited 0 over a refused configuration:\n%s", out)
	}
	requireContains(t, out, "keeps serving the previous one", "proxy:reload")
	if strings.Contains(out, "not running") {
		t.Errorf("proxy:reload says the proxy is not running:\n%s", out)
	}

	out, err = p.tryRun(10*time.Minute, "rebuild")
	if err == nil {
		t.Errorf("rebuild exited 0 over a refused configuration:\n%s", out)
	}
	requireContains(t, out, "did not take the new configuration", "rebuild")

	// Back to something nginx takes, for the teardown.
	p.run(time.Minute, "config:set", "--global", "-n", "proxy/worker/processes", "-v", "2")
}
