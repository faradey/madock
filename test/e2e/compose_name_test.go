//go:build e2e

package e2e

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// requireStatusOne fails unless the command ended with exit status 1.
func requireStatusOne(t *testing.T, err error, out, what string) {
	t.Helper()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 1 {
		t.Fatalf("%s: exit %v, want status 1\n%s", what, err, out)
	}
}

// A setup refused for its directory name printed the refusal and exited 0, so a
// script took it for a setup made.
func TestSetupRefusesAPeriodWithStatusOne(t *testing.T) {
	p := newProject(t, "e2e.period")

	out, err := p.tryRun(setupTimeout, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2eperiod.test",
	)

	requireStatusOne(t, err, out, "setup in a directory with a period")
	requireContains(t, out, "period", "the refusal should say why")
	if p.configured() {
		t.Errorf("a refused setup registered the project")
	}
}

// Compose lower-cases a project's name and drops what is not a-z, 0-9, '_' or
// '-', so E2eClash and e2eclash are one Compose project. madock took both: the
// second start recreated the first one's containers on its database volume,
// and the first one's status still said running. Reproduced on a VM before the
// fix. The second setup is refused, and so is a clone under such a name.
func TestAProjectComposeWouldShareIsRefused(t *testing.T) {
	install := newInstallation(t)

	first := install.project("E2eClash")
	first.run(setupTimeout, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2eclash1.test",
	)

	second := install.project("e2eclash")
	out, err := second.tryRun(setupTimeout, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2eclash2.test",
	)
	requireStatusOne(t, err, out, "setup of a second project with the same Compose name")
	requireContains(t, out, "E2eClash", "the refusal should name the project already there")
	if second.configured() {
		t.Errorf("the refused setup registered %s", second.name)
	}

	out, err = first.tryRun(5*time.Minute, "project:clone", "-n", "E2ECLASH", "-s=-copy")
	requireStatusOne(t, err, out, "project:clone under a name Compose shares with the source")
	requireContains(t, out, "madock_e2eclash", "the refusal should name the Compose project")
	if _, statErr := os.Stat(filepath.Join(first.execDir, "projects", "E2ECLASH")); !os.IsNotExist(statErr) {
		t.Errorf("the refused clone left projects/E2ECLASH behind")
	}
}

// Without its compose file a project is removed by its Compose label. The label
// of E2eLabel is madock_e2elabel, the lookup asked for madock_E2eLabel, found
// nothing, and the removal reported success with every container and volume
// still there. Reproduced on a VM before the fix, through project:purge.
func TestRemovalWithoutTheComposeFileFindsAMixedCaseProject(t *testing.T) {
	install := newInstallation(t)
	p := install.project("E2eLabel")

	p.run(setupTimeout, "setup", "-y",
		"--platform=custom",
		"--language=none",
		"--hosts=e2elabel.test",
	)
	p.run(20*time.Minute, "start")

	label := "label=com.docker.compose.project=madock_e2elabel"
	if len(dockerIDs(t, "ps", "-aq", "--filter", label)) == 0 {
		t.Fatalf("no container carries %s after start, so there is nothing to look for", label)
	}

	for _, file := range []string{"docker-compose.yml", "docker-compose.override.yml"} {
		if err := os.Remove(p.generated(file)); err != nil && !os.IsNotExist(err) {
			t.Fatalf("removing %s: %v", file, err)
		}
	}

	install.destroyProxy(p)
	p.run(5*time.Minute, "project:remove", "--force", "--name="+p.name)

	if left := dockerIDs(t, "ps", "-aq", "--filter", label); len(left) > 0 {
		t.Errorf("containers of %s survived its removal: %v", p.name, left)
	}
	if left := dockerIDs(t, "volume", "ls", "-q", "--filter", label); len(left) > 0 {
		t.Errorf("volumes of %s survived its removal: %v", p.name, left)
	}
}

func dockerIDs(t *testing.T, args ...string) []string {
	t.Helper()
	out, err := exec.Command("docker", args...).CombinedOutput()
	if err != nil {
		t.Fatalf("docker %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.Fields(string(out))
}
