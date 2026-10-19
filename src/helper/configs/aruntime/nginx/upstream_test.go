package nginx_test

import (
	"strings"
	"testing"

	"github.com/faradey/madock/v4/src/helper/configs/aruntime/nginx"
	"github.com/faradey/madock/v4/src/helper/docker"
)

// With ports published — the default, and every project before the key — the
// upstream is exactly what it always was.
func TestAPublishingProjectKeepsItsHostUpstream(t *testing.T) {
	for _, conf := range []map[string]string{{}, {"ports/publish": "true"}} {
		server, resolve := nginx.MainUpstream(conf, "shop", 17003)
		if server != "host.docker.internal:17003" || resolve != "" {
			t.Errorf("ports/publish %q: upstream %q resolve %q", conf["ports/publish"], server, resolve)
		}
	}
}

// Without published ports the proxy has nothing on the host to reach, so it
// goes to the container by name, at the port inside it — varnish's when varnish
// fronts the site — and resolves the name at run time.
func TestAProjectWithoutPortsIsReachedByContainerName(t *testing.T) {
	conf := map[string]string{"ports/publish": "false", "container_name_prefix": "madock-", "activeScope": "default"}
	server, resolve := nginx.MainUpstream(conf, "Shop", 17003)
	if want := docker.GetContainerName(conf, "Shop", "nginx") + ":80"; server != want {
		t.Errorf("upstream %q, want %q", server, want)
	}
	if resolve == "" {
		t.Errorf("an upstream named by container must be resolved at run time, or a stopped project stops the proxy")
	}

	conf["varnish/enabled"] = "true"
	conf["activeScope"] = "Staging"
	server, _ = nginx.MainUpstream(conf, "Shop", 17003)
	if want := docker.GetContainerName(conf, "Shop", "varnish") + ":6081"; server != want {
		t.Errorf("with varnish the upstream is %q, want %q", server, want)
	}
}

func TestTheResolverIsAddedOnlyWhenAnUpstreamResolves(t *testing.T) {
	if got := nginx.ResolverFor("upstream a {\n  server host.docker.internal:17003 max_fails=3 fail_timeout=30s;\n}\n"); got != "" {
		t.Errorf("a proxy with no named upstream gained a resolver:\n%s", got)
	}
	got := nginx.ResolverFor("upstream a {\n  zone a 64k;\n  server madock-shop-nginx-1:80 resolve max_fails=3 fail_timeout=30s;\n}\n")
	if !strings.Contains(got, "resolver 127.0.0.11") {
		t.Errorf("a named upstream got no resolver: %q", got)
	}
}
