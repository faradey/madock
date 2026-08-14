package nginx

import (
	"flag"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"

	"github.com/faradey/madock/v3/src/helper/testenv"
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

// allocatedPort matches the ports madock hands out, which are five digits from
// 17000 up. The proxy's own listeners — 80, 443, 35729, 5173 — are fixed and stay
// as they are, which is why the pattern is anchored on the range rather than on
// "any number".
var allocatedPort = regexp.MustCompile(`\b1[78]\d{3}\b`)

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
