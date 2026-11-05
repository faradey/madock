package nginx

import (
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/faradey/madock/v4/src/helper/paths"
	"github.com/faradey/madock/v4/src/helper/testenv"
)

// Run with -count=1: the input is docker/, plain data files the test cache knows
// nothing about, so without it editing a template and re-running reports `ok
// (cached)` and proves nothing.
//
//	go test -count=1 ./src/helper/configs/aruntime/nginx/... -run Golden -update
var updateGolden = flag.Bool("update", false, "rewrite golden files")

// TestGoldenProxyConf pins the configuration of the shared proxy — the file every
// project's traffic passes through, and until now the one generated file nothing
// covered at all.
//
// Three defects found in August lived in it and were each found by hand: a literal
// `Connection: upgrade` sent on every request, `Host $host` dropping the port on
// the way to the project, and `worker_priority -10` that a container cannot apply,
// logging two alerts per start for years. The generator is three hundred lines of
// branching, and a unit test on the preamble covers the first ten.
//
// It renders one project, because the interesting part is not how many blocks there
// are but what each contains: the upstreams, the http and https servers, the
// forwarded headers, the tool subpaths, and the livereload and vite listeners that
// only exist here.
func TestGoldenProxyConf(t *testing.T) {
	projectName := "golden"
	env := testenv.SetupWith(t, projectName, "golden.test", nil)

	MakeConf(projectName)

	rendered := testenv.Collect(t, filepath.Join(env.ExecDir, "aruntime", "ctx"), env)
	if len(rendered) == 0 {
		t.Fatal("MakeConf produced no proxy configuration")
	}
	if _, ok := rendered["proxy.conf"]; !ok {
		t.Fatalf("proxy.conf is not among the generated files: %v", keys(rendered))
	}

	// Only proxy.conf: the certificate and the key beside it are regenerated on
	// every run and are not text a golden file can hold.
	//
	// Port numbers go too, and that one cost a red CI run. Allocation skips ports
	// already in use on the machine, so this laptop's registry handed out 17003
	// where a fresh runner handed out 17000 — and the proxy configuration carries
	// the number in every upstream name and every proxy_pass. The names still have
	// to agree with the passes, so the substitution is by position: the first
	// distinct port becomes <PORT1>, the second <PORT2>, and a block that pointed
	// at the wrong one would still show up as a difference.
	only := map[string]string{"proxy.conf": maskPorts(rendered["proxy.conf"])}

	goldenDir := filepath.Join("testdata", "golden", "proxy")
	if *updateGolden {
		testenv.WriteGolden(t, goldenDir, only)
		return
	}
	testenv.CompareGolden(t, goldenDir, only)
}

// TestGoldenProxyConfWithoutPublishedPorts pins what ports/publish false renders:
// a name looked up per request and no upstream block for the project's web
// server. That block is what kept a stopped project's address and sent its
// requests to whichever project took it; see MainUpstream.
func TestGoldenProxyConfWithoutPublishedPorts(t *testing.T) {
	projectName := "golden"
	env := testenv.SetupWith(t, projectName, "golden.test", map[string]string{"ports/publish": "false"})

	MakeConf(projectName)

	rendered := testenv.Collect(t, filepath.Join(env.ExecDir, "aruntime", "ctx"), env)
	only := map[string]string{"proxy.conf": maskPorts(rendered["proxy.conf"])}

	goldenDir := filepath.Join("testdata", "golden", "proxy-no-published-ports")
	if *updateGolden {
		testenv.WriteGolden(t, goldenDir, only)
		return
	}
	testenv.CompareGolden(t, goldenDir, only)
}

// The proxy's network namespace changes only on a machine that needs it: a sysctl
// a Docker refuses would stop the proxy, and with it every project behind it.
func TestTheProxyReusesPortsOnlyForAProjectReachedByName(t *testing.T) {
	for publish, want := range map[string]bool{"true": false, "false": true} {
		testenv.SetupWith(t, "golden", "golden.test", map[string]string{"ports/publish": publish})

		MakeConf("golden")

		compose, err := os.ReadFile(paths.ProxyDockerCompose())
		if err != nil {
			t.Fatal(err)
		}
		if got := strings.Contains(string(compose), "net.ipv4.tcp_tw_reuse=1"); got != want {
			t.Errorf("ports/publish %s: tcp_tw_reuse in the proxy's compose is %v, want %v:\n%s", publish, got, want, compose)
		}
	}
}

// allocatedPort matches the ports madock hands out, which are five digits from
// 17000 up. The proxy's own listeners — 80, 443, 35729, 5173 — are fixed and stay
// as they are, which is why the pattern is anchored on the range rather than on
// "any number".
//
// No \b around it, and that is the point: the number appears as
// `upstreamm_madock_17003`, where an underscore is a word character, so a word
// boundary never falls between it and the first digit. The first version of this
// pattern had them, matched only the `host.docker.internal:17003` form, and left
// every upstream name with a live port in it — and the grep that was supposed to
// catch that used the same pattern, so it agreed. Go's regexp has no lookbehind to
// express it more tightly; the five-digit range is distinctive enough here.
var allocatedPort = regexp.MustCompile(`1[78]\d{3}`)

func maskPorts(content string) string {
	seen := map[string]string{}
	return allocatedPort.ReplaceAllStringFunc(content, func(port string) string {
		if name, ok := seen[port]; ok {
			return name
		}
		name := "<PORT" + strconv.Itoa(len(seen)+1) + ">"
		seen[port] = name
		return name
	})
}

func keys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
