//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestAProjectTemplateReadingAnUnknownSettingIsNamed: a project's own template
// asking for a setting madock does not have renders it empty, and until now did
// so without a word. Three projects on one developer machine read the port
// arithmetic madock dropped years ago and published ":80" with no host port.
// The start has to say so — and still start, because those projects run today.
func TestAProjectTemplateReadingAnUnknownSettingIsNamed(t *testing.T) {
	p := newProject(t, "e2eunknownkey")
	p.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2eunknownkey.test",
	)

	dir := filepath.Join(p.runDir, ".madock", "docker", "nginx", "vhost.d")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	snippet := "# retired: {{{.nginx.port.project}}}\n"
	if err := os.WriteFile(filepath.Join(dir, "probe.conf"), []byte(snippet), 0o644); err != nil {
		t.Fatal(err)
	}

	out := p.run(20*time.Minute, "start")
	if !strings.Contains(out, "reads settings this madock does not have") || !strings.Contains(out, "nginx/port/project") {
		t.Errorf("start rendered a template with an unknown setting and said nothing:\n%s", out)
	}
}
