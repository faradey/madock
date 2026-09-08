package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/faradey/madock/v4/src/helper/testenv"
)

// readGenerated reads one file out of the project's generated runtime.
func readGenerated(t *testing.T, env *testenv.Env, relative string) string {
	t.Helper()

	path := filepath.Join(env.ExecDir, "aruntime", "projects", env.ProjectName, relative)
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(body)
}

// The php limits reach the vhost, which is the half the golden fixtures cannot
// show: they render the defaults, and the defaults are the numbers that were
// hardcoded before. What had to be proved is that a project can change them.
//
// They were three copies of one number inside the platform vhosts, and the only
// way to change them was to copy a 240-line vhost into the project's own
// .madock/docker and let it drift from upstream. What runs into them is an
// import or a reindex through the web, and what that looks like is a 500 with
// nothing in the nginx log to explain it.
func TestPhpLimitsReachTheVhost(t *testing.T) {
	env := testenv.SetupWith(t, "limitsproject", "limits.test", map[string]string{
		"php/limits/memory":                 "4096M",
		"php/limits/max_execution_time":     "7200",
		"php/limits/max_execution_time_web": "900",
	})

	MakeConf("limitsproject")

	vhost := readGenerated(t, env, "ctx/nginx.conf")
	for _, want := range []string{
		"memory_limit=4096M",
		"max_execution_time=7200",
		"max_execution_time=900",
	} {
		if !strings.Contains(vhost, want) {
			t.Errorf("missing %q in the generated vhost:\n%s", want, firstLines(vhost, 60))
		}
	}

	if strings.Contains(vhost, "memory_limit=756M") {
		t.Errorf("the vhost still carries the old hardcoded memory limit:\n%s", firstLines(vhost, 60))
	}
}

// The body size a project accepts is its own setting now. Two limits sit in the
// path — this one and proxy/max_body_size in the shared proxy — and an upload
// that dies between them says nothing useful, which is why they are named the
// same thing in both places.
func TestProjectBodySizeReachesTheVhost(t *testing.T) {
	env := testenv.SetupWith(t, "bodysizeproject", "bodysize.test", map[string]string{
		"nginx/max_body_size": "512M",
	})

	MakeConf("bodysizeproject")

	vhost := readGenerated(t, env, "ctx/nginx.conf")
	if !strings.Contains(vhost, "client_max_body_size       512M;") {
		t.Errorf("the configured body size did not reach the vhost:\n%s", firstLines(vhost, 60))
	}
}

func firstLines(text string, n int) string {
	lines := strings.Split(text, "\n")
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n")
}

// The search engine's memory is the project's to set, and both numbers are
// needed rather than one.
//
// They were `memory: 2512m` and `-Xms800m -Xmx800m` inside the compose
// snippets, so every project paid the same regardless of catalogue size.
// Measured on extmag.com on 2026-09-09: java held 1650 MB of RSS against
// 12.4 MB of indices and 21 products, on a machine with 5.8 GB. The only way
// to change it was to copy the snippet into the project's own .madock/docker,
// which is a copy that then drifts from the shipped one in silence — and that
// copy existed on that machine when this was written.
//
// Two keys, because the container limit and the JVM heap are not the same
// number and cannot be derived from each other safely: the heap has to leave
// room for lucene's off-heap memory, and a limit below the heap makes the
// kernel kill the container instead of java throwing.
func TestSearchMemoryReachesTheComposeFile(t *testing.T) {
	for _, engine := range []string{"opensearch", "elasticsearch"} {
		other := "elasticsearch"
		if engine == "elasticsearch" {
			other = "opensearch"
		}

		t.Run(engine, func(t *testing.T) {
			// The other engine is switched off explicitly: this platform ships
			// both blocks, so leaving it on renders a second service carrying
			// the defaults, and the assertion below — that the old numbers are
			// gone — would fail against the neighbour rather than the code.
			env := testenv.SetupWith(t, "search"+engine, engine+".test", map[string]string{
				"search/engine":                      engine,
				"search/" + engine + "/enabled":      "true",
				"search/" + engine + "/memory_limit": "768m",
				"search/" + engine + "/heap":         "256m",
				"search/" + other + "/enabled":       "false",
			})

			MakeConf("search" + engine)

			compose := readGenerated(t, env, "docker-compose.yml")
			for _, want := range []string{"memory: 768m", "-Xms256m -Xmx256m"} {
				if !strings.Contains(compose, want) {
					t.Errorf("%s: missing %q in the generated compose file:\n%s",
						engine, want, firstLines(compose, 80))
				}
			}

			// The numbers that used to be hardcoded, named so that a template
			// which ignores the setting fails here rather than passing on a
			// substring that happens to match.
			for _, gone := range []string{"memory: 2512m", "-Xms800m -Xmx800m"} {
				if strings.Contains(compose, gone) {
					t.Errorf("%s: the compose file still carries the hardcoded %q", engine, gone)
				}
			}
		})
	}
}
