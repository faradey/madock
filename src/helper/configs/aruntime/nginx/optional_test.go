package nginx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	configs2 "github.com/faradey/madock/v4/src/helper/configs"
	"github.com/faradey/madock/v4/src/helper/ports"
	"github.com/faradey/madock/v4/src/helper/testenv"
)

// A project with nginx/enabled=false disappears from everything the shared proxy
// owns: the server block, the certificate, and the two reserved ports.
//
// Removing the project's <hosts> was the obvious way to ask for this and did
// none of it. The container still started; the proxy still wrote a block,
// because a project with no hosts gets loc.<name>.com invented for it, so the
// block was renamed rather than removed; and the ports stayed reserved against
// every other project on the machine. Each of the three is asserted here
// separately, because each was separately wrong.
func TestProjectWithoutNginxIsNotRouted(t *testing.T) {
	projectName := "golden"
	env := testenv.SetupWith(t, projectName, "golden.test", map[string]string{
		"nginx/enabled": "false",
	})

	MakeConf(projectName)

	proxyConf := read(t, filepath.Join(env.ExecDir, "aruntime", "ctx", "proxy.conf"))
	if strings.Contains(proxyConf, "golden.test") {
		t.Error("proxy.conf still routes the project's host")
	}
	// The fallback name is the one that made removing <hosts> useless: it is
	// invented when a project has none, so a block that looks gone is not.
	if strings.Contains(proxyConf, "loc."+projectName+".com") {
		t.Error("proxy.conf carries the invented fallback host for a project with no web server")
	}
	if strings.Contains(proxyConf, "upstreamm_madock") {
		t.Error("proxy.conf declares an upstream for a project with no web server")
	}

	if allocated := ports.GetRegistry().GetAllForProject(projectName); len(allocated) > 0 {
		t.Errorf("ports were reserved for a project with no web server: %v", allocated)
	}

	if names := sslAltNamesExt(); strings.Contains(names, "golden.test") {
		t.Error("the certificate would cover a host nothing serves")
	}
}

// The cache is what a project keeps while it is stopped, so a block that was
// written before the switch was thrown has to be deleted rather than skipped —
// otherwise the next start of any other project puts it back into proxy.conf.
// That is how a server_name for a project whose hosts had been removed survived
// on a dev machine for weeks.
func TestSwitchingNginxOffDropsTheCachedBlock(t *testing.T) {
	projectName := "golden"
	env := testenv.SetupWith(t, projectName, "golden.test", nil)

	MakeConf(projectName)

	cached := filepath.Join(env.ExecDir, "cache", projectName+"-proxy.conf")
	if _, err := os.Stat(cached); err != nil {
		t.Fatalf("the project was not cached in the first place: %v", err)
	}

	// The same installation, the same project, one setting changed — which is
	// what a user does. Setting it up a second time would build a fresh
	// temporary directory with no cache in it, and the test would pass without
	// ever exercising the deletion.
	configs2.SetParam(projectName, "nginx/enabled", "false", "default", configs2.MadockLevelConfigCode)
	configs2.CleanCache()

	MakeConf(projectName)

	if _, err := os.Stat(cached); !os.IsNotExist(err) {
		t.Errorf("the cached block survived the switch being turned off: %v", err)
	}

	proxyConf := read(t, filepath.Join(env.ExecDir, "aruntime", "ctx", "proxy.conf"))
	if strings.Contains(proxyConf, "golden.test") {
		t.Error("proxy.conf still routes the project after its web server was switched off")
	}
}

// The other side, and the one that matters more: a project with the default
// configuration is unaffected. This change should be invisible everywhere except
// where it was asked for.
func TestDefaultProjectStillRouted(t *testing.T) {
	projectName := "golden"
	env := testenv.SetupWith(t, projectName, "golden.test", nil)

	MakeConf(projectName)

	proxyConf := read(t, filepath.Join(env.ExecDir, "aruntime", "ctx", "proxy.conf"))
	if !strings.Contains(proxyConf, "golden.test") {
		t.Error("proxy.conf does not route a project that has a web server")
	}
	if ports.GetRegistry().Get(projectName, ports.ServiceNginx) == 0 {
		t.Error("no port was reserved for a project that has a web server")
	}
}

func read(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(body)
}
