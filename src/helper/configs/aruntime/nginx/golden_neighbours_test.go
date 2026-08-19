package nginx

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/faradey/madock/v3/src/helper/testenv"
)

// TestGoldenProxyWithTwoProjects pins the half of the shared proxy that one
// project cannot show: what happens when there is a neighbour.
//
// `proxy.conf` is a single file describing every project on the machine, and it
// is rebuilt whenever any one of them is generated. So the failure it can have
// that nothing else can is a project losing its block, or gaining somebody
// else's port — and against one project both are invisible, because there is
// only one of everything to compare.
//
// e2e covers the behaviour and cannot replace this. It needs docker, takes
// minutes, and reports "the neighbour is unreachable" rather than which line
// moved; a golden answers with the line.
func TestGoldenProxyWithTwoProjects(t *testing.T) {
	first := testenv.SetupWith(t, "goldenfirst", "goldenfirst.test", nil)
	MakeConf(first.ProjectName)

	second := testenv.AddProject(t, first, "goldensecond", "goldensecond.test", nil)
	MakeConf(second.ProjectName)

	rendered := testenv.Collect(t, filepath.Join(first.ExecDir, "aruntime", "ctx"), first)
	conf, ok := rendered["proxy.conf"]
	if !ok {
		t.Fatalf("proxy.conf is not among the generated files: %v", keys(rendered))
	}

	// Before the fixture: both projects are in there at all. A golden written
	// over a file that had lost a project would pin the loss.
	for _, host := range []string{"goldenfirst.test", "goldensecond.test"} {
		if !strings.Contains(conf, host) {
			t.Fatalf("%s is missing from the shared proxy:\n%s", host, conf)
		}
	}

	// The second project's run directory is its own, and it appears nowhere in a
	// proxy configuration — but normalising it costs nothing and means a future
	// template that does mention it cannot bake this machine's path into the
	// fixture.
	conf = strings.ReplaceAll(conf, second.RunDir, "<RUN_DIR_2>")

	only := map[string]string{"proxy.conf": maskPorts(conf)}

	goldenDir := filepath.Join("testdata", "golden", "proxy-two-projects")
	if *updateGolden {
		testenv.WriteGolden(t, goldenDir, only)
		return
	}
	testenv.CompareGolden(t, goldenDir, only)
}
