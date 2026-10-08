//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestAProjectNeedingANewerMadockIsRefused is the guard for a committed
// .madock/config.xml shared by developers on different madock versions: the
// project says which madock it needs, and an older one refuses instead of
// misreading what it does not know. The check sits in the dispatcher, so the
// witness is a real command, not the function.
func TestAProjectNeedingANewerMadockIsRefused(t *testing.T) {
	p := newProject(t, "e2eminver")
	p.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2eminver.test",
	)

	dir := filepath.Join(p.runDir, ".madock")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	config := `<?xml version="1.0" encoding="UTF-8"?>
<config>
    <scopes>
        <default>
            <madock>
                <min_version>99.0.0</min_version>
            </madock>
        </default>
    </scopes>
</config>
`
	if err := os.WriteFile(filepath.Join(dir, "config.xml"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := p.tryRun(2*time.Minute, "status")
	if err == nil {
		t.Fatalf("status ran for a project that needs madock 99.0.0:\n%s", out)
	}
	if !strings.Contains(out, "needs madock 99.0.0 or newer") {
		t.Errorf("the refusal should name the version the project needs:\n%s", out)
	}

	// The way out has to keep working: a person needs to see what is installed.
	if out, err := p.tryRun(2*time.Minute, "version"); err != nil {
		t.Errorf("version is refused too, which leaves no way to see what is installed: %v\n%s", err, out)
	}
}
