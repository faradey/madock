//go:build e2e

package e2e

import (
	"strings"
	"testing"
	"time"
)

// TestProjectLifecycle walks the path every madock user walks on their first
// day: create a project, start it, ask whether it is running, stop it.
//
// The platform is `custom` with no language, which is the cheapest shape there
// is — no PHP image to build, no Magento to install. That is deliberate. This
// test is not about Magento; it is about whether the commands that everything
// else is built on actually work end to end. A failure here means nothing else
// in the suite is worth reading.
func TestProjectLifecycle(t *testing.T) {
	p := newProject(t, "e2elife")

	out := p.run(3*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2elife.test",
	)
	t.Logf("setup said:\n%s", out)

	requireFile(t, p.execDir+"/projects/e2elife/config.xml", "setup should register the project")
	requireFile(t, p.generated("docker-compose.yml"), "setup should generate a compose file")

	// Generous: the first start of the first test pulls base images over the
	// network. Later runs hit the local image cache and take seconds.
	p.run(20*time.Minute, "start")

	// A command that exits 0 is not the same as a project that runs. Ask.
	services := []string{"app", "db", "nginx"}

	status := p.run(3*time.Minute, "status")
	for _, service := range services {
		requireContains(t, status, service+" running", "after start, "+service)
	}

	p.run(5*time.Minute, "stop")

	// Per service rather than a search for "running": the tools section of the
	// output says "Cron is not running", so anything looser passes for the
	// wrong reason.
	stopped := p.run(3*time.Minute, "status")
	for _, service := range services {
		if strings.Contains(stopped, service+" running") {
			t.Errorf("after stop, %s is still reported as running:\n%s", service, stopped)
		}
	}
}
