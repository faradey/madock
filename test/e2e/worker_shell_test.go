//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestAWorkerCommandWithShellSyntaxRunsWhole — a worker's command ran as
// `sh -c "exec <command>"`, so `a; b` and `a && b` stopped after a, and
// `set -a; . ./.env; …` failed on `exec set`: shell syntax worked only when
// wrapped in a second `sh -c` by hand (Indemorio IND-11). Every part of such a
// command now runs, and the worker stays up on its last one.
func TestAWorkerCommandWithShellSyntaxRunsWhole(t *testing.T) {
	p := newProject(t, "e2eworkershell")
	p.run(setupTimeout, "setup", "-y", "--platform=custom", "--language=none", "--hosts=e2eworkershell.test")
	if err := os.WriteFile(filepath.Join(p.runDir, ".env"), []byte("E2E_GREETING=from-env\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(p.runDir, ".madock")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	config := `<?xml version="1.0" encoding="UTF-8"?>
<config>
    <scopes>
        <default>
            <worker>
                <enabled>true</enabled>
                <programs>
                    <shell>set -a; . /var/www/html/.env; echo "$E2E_GREETING" &gt; /tmp/w.txt &amp;&amp; echo second &gt;&gt; /tmp/w.txt; sleep 3600</shell>
                </programs>
            </worker>
        </default>
    </scopes>
</config>
`
	if err := os.WriteFile(filepath.Join(dir, "config.xml"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	p.run(20*time.Minute, "start")

	var out string
	for i := 0; i < 20; i++ {
		out, _ = p.tryRun(time.Minute, "cli", "--service", "worker-shell", "cat", "/tmp/w.txt")
		if strings.Contains(out, "second") {
			break
		}
		time.Sleep(3 * time.Second)
	}
	requireContains(t, out, "from-env", "the .env sourced by set -a; . ./.env")
	requireContains(t, out, "second", "the part after &&")
	requireContains(t, p.run(time.Minute, "status"), "worker-shell running", "the worker staying up on its last command")
}
