//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestProjectRemoveLeavesNothingBehind checks that removing a project actually
// removes it.
//
// Leftovers here are all of the invisible kind. A directory nobody deletes is
// noticed by nobody; a port kept in the registry is a port no other project can
// have; and a volume that outlives its project sits on the disk until the disk
// is the problem. None of it produces an error message on the day it happens.
//
// The volume is the interesting one, and there is no madock command that lists
// volumes to ask with. So the test asks the only question that matters instead:
// it builds a new project under the same name — which gives it the same volume
// name — and looks for the old data. If the volume survived, yesterday's
// database is now in today's project, which is worse than a full disk.
func TestProjectRemoveLeavesNothingBehind(t *testing.T) {
	install := newInstallation(t)
	p := install.project("e2egone")

	p.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2egone.test",
	)
	p.run(20*time.Minute, "start")

	p.freshTable("ghost", "(note VARCHAR(32))")
	p.query("INSERT INTO ghost VALUES ('from-the-first-project')")

	// A port was allocated during setup; it should not stay reserved.
	portsFile := filepath.Join(p.execDir, "aruntime", "ports.conf")
	requireContains(t, readFile(t, portsFile), "e2egone", "the project should hold ports while it exists")

	p.run(5*time.Minute, "project:remove", "--force", "--name=e2egone")

	for _, gone := range []struct {
		path string
		what string
	}{
		{filepath.Join(p.execDir, "projects", "e2egone"), "the project registry entry"},
		{filepath.Join(p.execDir, "aruntime", "projects", "e2egone"), "the generated runtime"},
		{p.runDir, "the project directory"},
	} {
		if _, err := os.Stat(gone.path); !os.IsNotExist(err) {
			t.Errorf("%s is still there: %s", gone.what, gone.path)
		}
	}

	if content := readFile(t, portsFile); strings.Contains(content, "e2egone") {
		t.Errorf("the removed project still holds ports:\n%s", content)
	}

	// The same name again, which means the same container and volume names.
	// Anything Docker kept comes back here.
	second := install.project("e2egone")
	second.run(5*time.Minute, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2egone.test",
	)
	second.run(20*time.Minute, "start")

	tables := second.query("SHOW TABLES")
	if strings.Contains(tables, "ghost") {
		t.Errorf("the removed project's database volume survived and is now in its replacement:\n%s", tables)
	}
}
