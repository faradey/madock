package docker

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strings"

	"github.com/faradey/madock/v3/src/helper/cli/attr"
	configs2 "github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/configs/aruntime/project"
	"github.com/faradey/madock/v3/src/helper/logger"
	"github.com/faradey/madock/v3/src/helper/paths"
)

// composeProjectName returns the compose project name madock uses when
// generating docker-compose.yml. Containers/volumes/networks created by
// `docker compose up` are labelled with this exact string under
// com.docker.compose.project, which lets us find them later without the
// compose file (e.g. after the project directory was already removed,
// or when a different madock binary handles the cleanup).
func composeProjectName(projectName string) string {
	return "madock_" + projectName
}

// forceRemoveByLabel cleans up containers (and optionally volumes,
// networks, and images) that carry the compose project label for
// `projectName`. Used as a fallback in Down/Kill when the compose file
// is gone — `docker compose down` cannot operate without it, but the
// docker resources themselves still exist and need to be removed.
// It is also safe to call after a successful `compose down`: compose
// has already removed everything it owned, so the queries below come
// back empty and the cleanup is a no-op.
func forceRemoveByLabel(projectName string, withVolumes bool) {
	labelFilter := "label=com.docker.compose.project=" + composeProjectName(projectName)

	// Containers first — they hold references to the network they sit on.
	if ids := dockerQuery("ps", "-aq", "--filter", labelFilter); len(ids) > 0 {
		dockerRun("rm", append([]string{"-f"}, ids...)...)
	}

	if withVolumes {
		if vols := dockerQuery("volume", "ls", "-q", "--filter", labelFilter); len(vols) > 0 {
			dockerRun("volume", append([]string{"rm", "-f"}, vols...)...)
		}
		if imgs := dockerQuery("images", "-q", "--filter", labelFilter); len(imgs) > 0 {
			dockerRun("rmi", append([]string{"-f"}, imgs...)...)
		}
	}

	if nets := dockerQuery("network", "ls", "-q", "--filter", labelFilter); len(nets) > 0 {
		dockerRun("network", append([]string{"rm"}, nets...)...)
	}
}

// dockerQuery runs `docker <subject> <args…>` and returns whitespace-
// separated tokens from stdout. Returns nil on any error so callers can
// just check `len(...) > 0` before acting.
func dockerQuery(subject string, args ...string) []string {
	out, err := exec.Command("docker", append([]string{subject}, args...)...).Output()
	if err != nil {
		return nil
	}
	return strings.Fields(strings.TrimSpace(string(out)))
}

// dockerRun runs `docker <subject> <args…>` and swallows errors. The
// cleanup helpers are best-effort — surfacing failures (e.g. a network
// that's still attached to something else) adds noise without giving
// the user anything actionable.
func dockerRun(subject string, args ...string) {
	_ = exec.Command("docker", append([]string{subject}, args...)...).Run()
}

// UpWithBuild starts both nginx proxy and project containers with build
func UpWithBuild(projectName string, withChown bool) {
	UpNginxWithBuild(projectName, true)
	UpProjectWithBuild(projectName, withChown)
}

// Down stops project containers. When the compose file is present it
// goes through `docker compose down`; when it isn't (e.g. the project
// dir was already removed, or another madock binary owns the compose
// files), it falls back to scanning docker by the compose project
// label so orphan containers/volumes/networks/images still get cleaned.
func Down(projectName string, withVolumes bool) {
	pp := paths.NewProjectPaths(projectName)
	composeFile := pp.DockerCompose()
	composeFileOS := pp.DockerComposeOverride()
	if paths.IsFileExist(composeFile) {
		profilesOn := []string{
			"compose",
			"-f",
			composeFile,
			"-f",
			composeFileOS,
		}

		profilesOn = append(profilesOn, "down")

		if withVolumes {
			profilesOn = append(profilesOn, "-v")
			profilesOn = append(profilesOn, "--rmi")
			profilesOn = append(profilesOn, "all")
		}

		cmd := exec.Command("docker", profilesOn...)
		attachOutput(cmd)
		err := cmd.Run()
		if err != nil {
			fmt.Println(err)
		}
	}

	// Label-based sweep — handles the no-compose-file case and also
	// catches any leftovers that compose missed.
	forceRemoveByLabel(projectName, withVolumes)
}

// ServiceState is one row of `docker compose ps -a` for a project.
type ServiceState struct {
	Service  string `json:"Service"`
	Name     string `json:"Name"`
	State    string `json:"State"`
	ExitCode int    `json:"ExitCode"`
}

// NotRunning returns the project's services that are not running.
//
// It exists because "started successfully" was being printed on the strength
// of `docker compose up` returning zero, which only says the containers were
// created. A container whose main process is not a daemon — a Node service
// running a command that exits, say — is gone seconds later, and until
// somebody read the log there was nothing to go on: start said success and
// status, asked in between, honestly said running.
func NotRunning(projectName string) []ServiceState {
	pp := paths.NewProjectPaths(projectName)
	composeFile := pp.DockerCompose()
	if !paths.IsFileExist(composeFile) {
		return nil
	}

	out, err := exec.Command("docker", "compose", "-f", composeFile, "-f", pp.DockerComposeOverride(),
		"ps", "-a", "--format", "json").Output()
	if err != nil {
		// Nothing to report rather than a false alarm: a docker that cannot be
		// asked is not evidence that a service died.
		return nil
	}

	var dead []ServiceState
	for _, entry := range parseComposePS(out) {
		if entry.State != "running" && entry.State != "restarting" {
			dead = append(dead, entry)
		}
	}
	return dead
}

// parseComposePS reads `docker compose ps --format json` in both shapes it
// comes in: NDJSON from newer compose, a single JSON array from older.
func parseComposePS(psOutput []byte) []ServiceState {
	var entries []ServiceState
	for _, line := range strings.Split(string(psOutput), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") {
			var batch []ServiceState
			if err := json.Unmarshal([]byte(line), &batch); err == nil {
				entries = append(entries, batch...)
			}
			continue
		}
		var one ServiceState
		if err := json.Unmarshal([]byte(line), &one); err == nil && one.Service != "" {
			entries = append(entries, one)
		}
	}
	return entries
}

// Stop stops the project's containers without removing them. Unlike Down it
// leaves the containers in place, so a later Start brings the project back
// without recreating or rebuilding anything — which is what a snapshot needs:
// the data directory has to be quiet, not gone.
func Stop(projectName string) error {
	pp := paths.NewProjectPaths(projectName)
	composeFile := pp.DockerCompose()
	if !paths.IsFileExist(composeFile) {
		return nil
	}
	cmd := exec.Command("docker", "compose", "-f", composeFile, "-f", pp.DockerComposeOverride(), "stop")
	attachOutput(cmd)
	return cmd.Run()
}

// Start starts containers that already exist. It does not create or rebuild
// anything — see UpProjectWithBuild for that.
func Start(projectName string) error {
	pp := paths.NewProjectPaths(projectName)
	composeFile := pp.DockerCompose()
	if !paths.IsFileExist(composeFile) {
		return nil
	}
	cmd := exec.Command("docker", "compose", "-f", composeFile, "-f", pp.DockerComposeOverride(), "start")
	attachOutput(cmd)
	return cmd.Run()
}

// Kill forcefully stops project containers. Falls back to a
// label-based `docker kill` if the compose file is missing.
func Kill(projectName string) {
	pp := paths.NewProjectPaths(projectName)
	composeFile := pp.DockerCompose()
	composeFileOS := pp.DockerComposeOverride()
	if paths.IsFileExist(composeFile) {
		profilesOn := []string{
			"compose",
			"-f",
			composeFile,
			"-f",
			composeFileOS,
		}

		profilesOn = append(profilesOn, "kill")

		cmd := exec.Command("docker", profilesOn...)
		attachOutput(cmd)
		err := cmd.Run()
		if err != nil {
			fmt.Println(err)
		}
		return
	}

	// No compose file — kill running containers by compose project label.
	labelFilter := "label=com.docker.compose.project=" + composeProjectName(projectName)
	if ids := dockerQuery("ps", "-q", "--filter", labelFilter); len(ids) > 0 {
		dockerRun("kill", ids...)
	}
}

// UpProjectWithBuild starts project containers with build
func UpProjectWithBuild(projectName string, withChown bool) {
	var err error
	globalComposer := paths.ComposerDir()
	if !paths.IsFileExist(globalComposer) {
		err = os.Chmod(paths.MakeDirsByPath(globalComposer), 0777)
		if err != nil {
			logger.Fatal(err)
		}
	}

	composerGlobalDir, err := os.UserHomeDir()
	if err != nil {
		logger.Fatal(err)
	} else {
		if !paths.IsFileExist(composerGlobalDir + "/.composer") {
			paths.MakeDirsByPath(composerGlobalDir + "/.composer")
		}
	}

	pp := paths.NewProjectPaths(projectName)
	src := paths.MakeDirsByPath(pp.ComposerDir())

	if fi, err := os.Lstat(src); err == nil {
		if fi.Mode()&os.ModeSymlink != os.ModeSymlink {
			err = os.RemoveAll(src)
			if err == nil {
				err = os.Symlink(composerGlobalDir+"/.composer", src)
				if err != nil {
					logger.Fatal(err)
				}
			} else {
				fmt.Println(err)
			}
		}
	} else {
		err = os.Symlink(composerGlobalDir+"/.composer", src)
		if err != nil {
			logger.Fatal(err)
		}
	}

	sshDir := pp.SSHDir()

	if fi, err := os.Lstat(sshDir); err == nil {
		if fi.Mode()&os.ModeSymlink != os.ModeSymlink {
			err = os.RemoveAll(sshDir)
			if err == nil {
				err = os.Symlink(composerGlobalDir+"/.ssh", sshDir)
				if err != nil {
					logger.Fatal(err)
				}
			} else {
				fmt.Println(err)
			}
		}
	} else {
		err = os.Symlink(composerGlobalDir+"/.ssh", sshDir)
		if err != nil {
			logger.Fatal(err)
		}
	}

	paths.MakeDirsByPath(pp.RuntimeDir())
	composeFile := pp.DockerCompose()
	composeFileOS := pp.DockerComposeOverride()
	profilesOn := []string{
		"compose",
		"-f",
		composeFile,
		"-f",
		composeFileOS,
		"up",
		"--build",
		"--force-recreate",
		"--no-deps",
		"-d",
	}
	dockerComposePull([]string{"compose", "-f", composeFile, "-f", composeFileOS})
	cmd := exec.Command("docker", profilesOn...)
	attachOutput(cmd)
	err = cmd.Run()
	if err != nil {
		logger.Fatal(err)
	}

	projectConf := configs2.GetProjectConfig(projectName)

	if val, ok := projectConf["cron/enabled"]; ok && val == "true" {
		CronExecute(projectName, true, false)
	} else {
		CronExecute(projectName, false, false)
	}

	if withChown {
		usr, _ := user.Current()
		// The service running the application code, not "php" — a Node, Python
		// or Go project has no php container, and reaching for one turned
		// --with-chown into a fatal error on those platforms.
		mainService := configs2.ResolveMainService(projectConf, "php")
		chownCmd := "chown -R " + usr.Uid + ":" + usr.Gid + " " + projectConf["workdir"]
		if mainService == "php" {
			// Only the php image mounts the composer home.
			chownCmd += " && chown -R " + usr.Uid + ":" + usr.Gid + " /var/www/.composer"
		}
		/* for .npm for futures +" && chown -R "+usr.Uid+":"+usr.Gid+" /var/www/.npm" */
		err = ContainerExec(GetContainerName(projectConf, projectName, mainService), "root", true, "bash", "-c", chownCmd)
		if err != nil {
			logger.Fatal(err)
		}
	}

	// The containers now match the generated stack. Recorded last, so a failed
	// up leaves the old record and the next start retries instead of assuming
	// the change went in.
	project.RecordApplied(projectName)
}

// dockerComposePull pulls images for docker-compose
func dockerComposePull(composeFiles []string) {
	composeFiles = append(composeFiles, "pull")
	if attr.IsQuiet {
		composeFiles = append(composeFiles, "--quiet")
	}
	cmd := exec.Command("docker", composeFiles...)
	attachOutput(cmd)
	err := cmd.Run()
	if err != nil {
		logger.Fatal(err)
	}
}

// attachOutput connects cmd stdout/stderr to os.Stdout/os.Stderr unless quiet mode is active
func attachOutput(cmd *exec.Cmd) {
	attr.AttachOutput(cmd)
}

// UpSnapshot starts snapshot container
func UpSnapshot(projectName string) {
	pp := paths.NewProjectPaths(projectName)
	paths.MakeDirsByPath(pp.RuntimeDir())
	composerFile := pp.DockerComposeSnapshot()
	profilesOn := []string{
		"compose",
		"-f",
		composerFile,
		"up",
		"--build",
		"--force-recreate",
		"--no-deps",
		"-d",
	}
	dockerComposePull([]string{"compose", "-f", composerFile})
	cmd := exec.Command("docker", profilesOn...)
	attachOutput(cmd)
	err := cmd.Run()
	if err != nil {
		logger.Fatal(err)
	}
}

// StopSnapshot stops snapshot container
func StopSnapshot(projectName string) {
	pp := paths.NewProjectPaths(projectName)
	composerFile := pp.DockerComposeSnapshot()
	if paths.IsFileExist(composerFile) {
		command := "stop"
		cmd := exec.Command("docker", "compose", "-f", composerFile, command)
		attachOutput(cmd)
		err := cmd.Run()
		if err != nil {
			fmt.Println(err)
		}
	}
}
