package docker

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/faradey/madock/v4/src/helper/cli/fmtc"
	configs2 "github.com/faradey/madock/v4/src/helper/configs"
	"github.com/faradey/madock/v4/src/helper/configs/aruntime/project"
	"github.com/faradey/madock/v4/src/helper/paths"
)

// ApplyStackDiff brings the running containers in line with the rendered
// stack, touching only the services the diff names.
//
// Order matters, and each step is placed for a reason:
//
//  1. Restarts first. A service whose mounted files changed re-reads them on
//     restart; nothing else depends on it having done so.
//  2. Recreates next, with the image rebuilt. A recreated container comes back
//     under a new address.
//  3. nginx reloads last, because of that new address: nginx resolves its
//     upstreams when it loads the configuration, not per request, so a
//     recreated php behind an nginx that was left alone is a 502 until nginx
//     reloads. The reload also picks up a changed vhost — the case this whole
//     mechanism was built for.
//
// A reload is preceded by `nginx -t`. A vhost that does not parse used to take
// nginx down with the recreate; now it is refused and the old configuration
// keeps serving, and the refusal is printed with nginx's own words.
//
// The record of what is running is written once, at the end. The caller runs
// this instead of recreating everything, so the diff it was given is exactly
// what has to succeed before the stack may be called applied.
func ApplyStackDiff(projectName string, diff project.StackDiff, withChown bool) error {
	pp := paths.NewProjectPaths(projectName)
	compose := []string{"compose", "-f", pp.DockerCompose(), "-f", pp.DockerComposeOverride()}

	var restart []string
	reloadNginx := false
	for _, service := range diff.Restart {
		if service == "nginx" {
			reloadNginx = true
			continue
		}
		restart = append(restart, service)
	}

	if len(restart) > 0 {
		fmtc.ToDoLn("Restarting " + strings.Join(restart, ", ") + " — mounted files changed")
		// A database that is still initialising must not be stopped: dbinit.go.
		WaitForDatabaseInit(projectName)
		cmd := exec.Command("docker", append(append([]string{}, compose...), append([]string{"restart"}, restart...)...)...)
		attachOutput(cmd)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("restart %s: %w", strings.Join(restart, ", "), err)
		}
	}

	if len(diff.Recreate) > 0 {
		fmtc.ToDoLn("Recreating " + strings.Join(diff.Recreate, ", ") + " — definition changed")
		upProjectWithBuild(projectName, withChown, false, diff.Recreate)
		// Anything recreated may now have a new address that nginx still
		// resolves to the old one.
		if !contains(diff.Recreate, "nginx") && nginxDeclared(projectName) {
			reloadNginx = true
		}
	}

	// A stopped nginx has nothing to reload and will read the new files when it
	// is started; only a running one holds an old configuration in memory.
	if reloadNginx && serviceRunning(projectName, "nginx") {
		if err := ReloadProjectNginx(projectName); err != nil {
			return err
		}
	}

	project.RecordApplied(projectName)
	return nil
}

func serviceRunning(projectName, service string) bool {
	states, err := ServiceStates(projectName)
	if err != nil {
		return false
	}
	for _, s := range states {
		if s.Service == service {
			return s.State == "running"
		}
	}
	return false
}

// ReloadProjectNginx makes the project's nginx re-read its configuration
// without a restart — no dropped connections, no new address, no window.
//
// The test comes first and is not optional: `nginx -s reload` on a broken
// configuration prints the error and keeps the old one, which reads as success
// to a caller checking the exit code of the reload alone.
func ReloadProjectNginx(projectName string) error {
	projectConf := configs2.GetProjectConfig(projectName)
	container := GetContainerName(projectConf, projectName, "nginx")

	test := exec.Command("docker", "exec", container, "nginx", "-t")
	if out, err := test.CombinedOutput(); err != nil {
		return fmt.Errorf("the new nginx configuration does not pass `nginx -t`; the old one keeps serving:\n%s", strings.TrimSpace(string(out)))
	}

	fmtc.ToDoLn("Reloading nginx")
	reload := exec.Command("docker", "exec", container, "nginx", "-s", "reload")
	if out, err := reload.CombinedOutput(); err != nil {
		return fmt.Errorf("nginx -s reload: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func nginxDeclared(projectName string) bool {
	return DeclaredServices(projectName)["nginx"]
}
