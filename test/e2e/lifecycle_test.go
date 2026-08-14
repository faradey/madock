//go:build e2e

package e2e

import (
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

	// Asked as JSON, and only about the project's own services.
	//
	// The text form has three sections — Services, Proxy, Tools — and the shared
	// proxy's container is also called nginx. Matching "<service> running" against
	// the whole output therefore passes or fails on the state of the proxy: this
	// test failed on a runner where a neighbouring test had left the proxy up, with
	// the project correctly stopped. Reading the section that was asked about takes
	// the ambiguity away entirely.
	projectServices := func(what string) map[string]bool {
		t.Helper()

		var payload struct {
			Data struct {
				Services []struct {
					Service string `json:"service"`
					Running bool   `json:"running"`
				} `json:"services"`
			} `json:"data"`
		}
		out := p.run(3*time.Minute, "status", "--json")
		decode(t, out, &payload, "status --json "+what)

		running := map[string]bool{}
		for _, service := range payload.Data.Services {
			running[service.Service] = service.Running
		}
		return running
	}

	afterStart := projectServices("after start")
	for _, service := range services {
		if !afterStart[service] {
			t.Errorf("after start, %s is not running: %v", service, afterStart)
		}
	}

	// The text form as well, because that is what a person reads — but only that it
	// mentions each service as running, which is true regardless of the proxy.
	status := p.run(3*time.Minute, "status")
	for _, service := range services {
		requireContains(t, status, service+" running", "after start, "+service)
	}

	p.run(5*time.Minute, "stop")

	afterStop := projectServices("after stop")
	for _, service := range services {
		if afterStop[service] {
			t.Errorf("after stop, %s is still running: %v", service, afterStop)
		}
	}
}
