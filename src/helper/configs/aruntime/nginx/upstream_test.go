package nginx_test

import (
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/faradey/madock/v4/src/helper/configs/aruntime/nginx"
	"github.com/faradey/madock/v4/src/helper/docker"
	"github.com/faradey/madock/v4/src/helper/paths"
	"github.com/faradey/madock/v4/src/helper/ports"
	"github.com/faradey/madock/v4/src/helper/testenv"
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
	for name, conf := range map[string]string{
		"a variable in proxy_pass": "location / {\n            set $madock_upstream_17003 madock-shop-nginx-1:80;\n            proxy_pass http://$madock_upstream_17003;\n}\n",
		"a project's own template": "upstream a {\n  zone a 64k;\n  server madock-shop-nginx-1:80 resolve max_fails=3 fail_timeout=30s;\n}\n",
	} {
		if got := nginx.ResolverFor(conf); !strings.Contains(got, "resolver 127.0.0.11") {
			t.Errorf("%s got no resolver: %q", name, got)
		}
	}
}

var (
	declaredUpstream = regexp.MustCompile(`upstream (upstreamm_madock_\d+) \{[^}]*?\n\s*server (\S+)`)
	passedUpstream   = regexp.MustCompile(`proxy_pass\s+http://(upstreamm_madock_\d+)`)
	setVariable      = regexp.MustCompile(`set (\$madock_upstream_\d+) (\S+);`)
	passedVariable   = regexp.MustCompile(`proxy_pass\s+http://(\$madock_upstream_\d+);`)
)

// A proxy_pass naming an upstream nothing declares is `[emerg] host not found in
// upstream`: nginx refuses the whole file, and that file is every project on the
// machine. With varnish on, the server is varnish's port and the block is still
// declared under nginx's, so a name built from the wrong one of the two renders
// without a complaint and only nginx objects.
func TestEveryUpstreamAProxyPassNamesIsDeclared(t *testing.T) {
	for _, varnish := range []string{"false", "true"} {
		t.Run("varnish "+varnish, func(t *testing.T) {
			testenv.SetupWith(t, "golden", "golden.test", map[string]string{"varnish/enabled": varnish})

			nginx.MakeConf("golden")

			proxyConf, err := os.ReadFile(paths.CtxDir() + "/proxy.conf")
			if err != nil {
				t.Fatal(err)
			}

			servers := map[string]string{}
			for _, match := range declaredUpstream.FindAllStringSubmatch(string(proxyConf), -1) {
				servers[match[1]] = match[2]
			}
			passes := passedUpstream.FindAllStringSubmatch(string(proxyConf), -1)
			if len(passes) == 0 {
				t.Fatalf("no proxy_pass to an upstream in:\n%s", proxyConf)
			}
			for _, match := range passes {
				if _, declared := servers[match[1]]; !declared {
					t.Errorf("proxy_pass names %s, declared are %v", match[1], servers)
				}
			}

			// Declared is not enough: the two main blocks have to reach what is
			// in front of the site.
			service := "nginx"
			if varnish == "true" {
				service = "varnish"
			}
			want := "host.docker.internal:" + strconv.Itoa(ports.GetPort("golden", service))
			main := "upstreamm_madock_" + strconv.Itoa(ports.GetPort("golden", "nginx"))
			if got := servers[main]; got != want {
				t.Errorf("%s holds server %q, want %q", main, got, want)
			}
			if got := strings.Count(string(proxyConf), "http://"+main+";"); got != 2 {
				t.Errorf("%d proxy_pass to %s, want 2: the http and the https block", got, main)
			}
		})
	}
}

// The same for a project reached by name: there is no upstream block, and a
// variable proxy_pass reads without a `set` is an unknown variable to nginx, which
// refuses the file just the same.
func TestEveryVariableAProxyPassReadsIsSet(t *testing.T) {
	conf := map[string]string{"ports/publish": "false", "varnish/enabled": "true"}
	testenv.SetupWith(t, "golden", "golden.test", conf)

	nginx.MakeConf("golden")

	proxyConf, err := os.ReadFile(paths.CtxDir() + "/proxy.conf")
	if err != nil {
		t.Fatal(err)
	}

	servers := map[string]string{}
	for _, match := range setVariable.FindAllStringSubmatch(string(proxyConf), -1) {
		servers[match[1]] = match[2]
	}
	passes := passedVariable.FindAllStringSubmatch(string(proxyConf), -1)
	if len(passes) != 2 {
		t.Fatalf("%d proxy_pass through a variable, want 2: the http and the https block", len(passes))
	}
	for _, match := range passes {
		server, set := servers[match[1]]
		if !set {
			t.Errorf("proxy_pass reads %s, set are %v", match[1], servers)
		}
		if !strings.Contains(server, "varnish") || !strings.HasSuffix(server, ":6081") {
			t.Errorf("%s holds %q, want the varnish container at 6081", match[1], server)
		}
	}
}

// The proxy's copy of the container name follows Compose's rule too: project
// Shop+1 runs as madock-shop1, and an upstream with the '+' names nothing.
func TestTheProxyNamesTheContainerComposeCreates(t *testing.T) {
	conf := map[string]string{"ports/publish": "false", "container_name_prefix": "madock-", "activeScope": "default"}
	server, _ := nginx.MainUpstream(conf, "Shop+1", 17003)
	if server != "madock-shop1-nginx-1:80" {
		t.Errorf("upstream %q, want madock-shop1-nginx-1:80", server)
	}
	if want := docker.GetContainerName(conf, "Shop+1", "nginx") + ":80"; server != want {
		t.Errorf("upstream %q, docker.GetContainerName gives %q", server, want)
	}
}
