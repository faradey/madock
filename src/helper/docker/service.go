package docker

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/faradey/madock/v4/src/helper/paths"
)

// ComposeServices returns the project's services under the names compose knows
// them by.
//
// Asked of compose rather than assembled from the config, and that is the whole
// point: the generated stack holds services no config key names — `deployer`,
// `worker-<program>`, `php_without_xdebug` — so a list built here would be a
// second spelling of something compose already answers, and it would go stale
// the first time a template grew a service. It is also the only honest thing to
// put in a "no such service" message.
func ComposeServices(projectName string) ([]string, error) {
	pp := paths.NewProjectPaths(projectName)
	composeFile := pp.DockerCompose()
	if !paths.IsFileExist(composeFile) {
		return nil, fmt.Errorf("the project has no generated stack yet (%s is missing) — run `madock start` first", composeFile)
	}

	out, err := exec.Command("docker", "compose", "-f", composeFile, "-f", pp.DockerComposeOverride(),
		"config", "--services").Output()
	if err != nil {
		return nil, composeError(err)
	}

	var services []string
	for _, line := range strings.Split(string(out), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			services = append(services, line)
		}
	}
	return services, nil
}

// RestartServices restarts the named services and leaves every other container
// of the project alone.
//
// The difference from `restart` is not a matter of scope but of what is
// possible. A deploy recipe cannot call `restart`: it stops the deployer
// container, which is the process running the recipe, so the recipe dies at the
// moment it succeeds. That is why restarting after a deploy stayed a second
// step for a person, and why on 2026-08-19 three of four projects on one
// machine were serving code from a release older than `current` — the symlink
// had moved and the process had not been told.
// Extension point for madock-pro: its post-deploy hook is the only caller, and
// that hook is the answer to the paragraph above — the step a person used to
// forget.
func RestartServices(projectName string, services []string) error {
	if len(services) == 0 {
		return errors.New("no services named")
	}

	pp := paths.NewProjectPaths(projectName)
	composeFile := pp.DockerCompose()
	if !paths.IsFileExist(composeFile) {
		return fmt.Errorf("the project has no generated stack yet (%s is missing) — run `madock start` first", composeFile)
	}

	args := append([]string{"compose", "-f", composeFile, "-f", pp.DockerComposeOverride(), "restart"}, services...)
	cmd := exec.Command("docker", args...)
	attachOutput(cmd)
	if err := cmd.Run(); err != nil {
		return err
	}

	// A container comes back running the image's CMD and nothing else, so
	// anything madock started *inside* it is gone — cron above all, since no
	// application image starts it. Rearming belongs here rather than in the
	// `service:restart` controller: the other caller is the restart a finished
	// deploy performs, which is where the scheduler was actually being lost.
	EnsureCronAfterRestart(projectName, services)
	return nil
}

// ServiceStates returns one row per container of the project, running or not.
//
// An error means the question could not be asked — a docker that does not
// answer — which is not the same as a project with no containers, and callers
// that round the two together are how "restarted successfully" gets printed
// over a service that never came back.
func ServiceStates(projectName string) ([]ServiceState, error) {
	pp := paths.NewProjectPaths(projectName)
	composeFile := pp.DockerCompose()
	if !paths.IsFileExist(composeFile) {
		return nil, fmt.Errorf("the project has no generated stack yet (%s is missing)", composeFile)
	}

	out, err := exec.Command("docker", "compose", "-f", composeFile, "-f", pp.DockerComposeOverride(),
		"ps", "-a", "--format", "json").Output()
	if err != nil {
		return nil, composeError(err)
	}
	return parseComposePS(out), nil
}

// composeError keeps docker's own words.
//
// exec.Command().Output() puts them in ExitError.Stderr, and an ExitError
// printed as `%v` reads "exit status 1" — a message that names neither the
// service nor the reason, on a command whose entire value is saying which of
// the two happened.
func composeError(err error) error {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if msg := strings.TrimSpace(string(exitErr.Stderr)); msg != "" {
			return fmt.Errorf("%w: %s", err, msg)
		}
	}
	return err
}
