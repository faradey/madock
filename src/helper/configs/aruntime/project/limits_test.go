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
