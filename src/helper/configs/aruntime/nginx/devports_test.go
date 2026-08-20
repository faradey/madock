package nginx

import (
	"os"
	"strings"
	"testing"

	"github.com/faradey/madock/v4/src/helper/paths"
	"github.com/faradey/madock/v4/src/helper/testenv"
)

// The shared proxy publishes two fixed development ports: LiveReload on 35729
// and Vite on 5173. Fixed means they fall outside the 17000-19999 range the
// firewall guard closes, so on a server they answer the internet — which is how
// a Vite dev server was found listening on a production host.
//
// They are also published whether or not anything can serve them: Vite needs
// nodejs, which a project may not run at all.
//
// On by default, because on a laptop that is the whole point of them.
func TestProxyPublishesTheDevPortsByDefault(t *testing.T) {
	testenv.SetupWith(t, "golden", "golden.test", nil)

	makeDockerCompose("golden")

	compose := readProxyCompose(t)
	for _, port := range []string{"35729:35729", "5173:5173"} {
		if !strings.Contains(compose, port) {
			t.Errorf("the proxy should publish %s by default:\n%s", port, compose)
		}
	}
}

// And each can be turned off on its own, which is what a server does.
func TestProxyDevPortsCanBeTurnedOff(t *testing.T) {
	testenv.SetupWith(t, "golden", "golden.test", map[string]string{
		"proxy/livereload/publish": "false",
		"proxy/vite/publish":       "false",
	})

	makeDockerCompose("golden")

	compose := readProxyCompose(t)
	for _, port := range []string{"35729", "5173"} {
		if strings.Contains(compose, port) {
			t.Errorf("%s is still published after being turned off:\n%s", port, compose)
		}
	}

	// The ports that carry the sites are not development ports and stay.
	for _, port := range []string{":80", ":443"} {
		if !strings.Contains(compose, port) {
			t.Errorf("the proxy stopped publishing %s:\n%s", port, compose)
		}
	}
}

// One off and one on: they are separate services and a machine may want the
// difference.
func TestProxyDevPortsAreIndependent(t *testing.T) {
	testenv.SetupWith(t, "golden", "golden.test", map[string]string{
		"proxy/vite/publish": "false",
	})

	makeDockerCompose("golden")

	compose := readProxyCompose(t)
	if !strings.Contains(compose, "35729") {
		t.Error("turning off vite should not take livereload with it")
	}
	if strings.Contains(compose, "5173") {
		t.Error("vite is still published after being turned off")
	}
}

func readProxyCompose(t *testing.T) string {
	t.Helper()

	body, err := os.ReadFile(paths.ProxyDockerCompose())
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}
