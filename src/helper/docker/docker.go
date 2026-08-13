package docker

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
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

// HasContainers reports whether the project has any containers at all, running
// or stopped.
//
// `docker compose start` wakes existing containers and succeeds with nothing to
// do when there are none — exit zero, no output, a fraction of a second. The
// start command took that for success, so a project whose containers had been
// removed reported as started and was not. It is how a fresh clone behaved:
// clone removes the containers to load the copied data, and the configuration
// fingerprint still matched, so nothing suggested recreating them.
//
// An unanswerable docker returns true: that is not evidence of an empty
// project, and creating containers on a false negative is the more expensive
// mistake.
func HasContainers(projectName string) bool {
	pp := paths.NewProjectPaths(projectName)
	composeFile := pp.DockerCompose()
	if !paths.IsFileExist(composeFile) {
		return false
	}

	out, err := exec.Command("docker", "compose", "-f", composeFile, "-f", pp.DockerComposeOverride(),
		"ps", "-a", "--format", "json").Output()
	if err != nil {
		return true
	}

	// The snapshot helper does not count. It belongs to the same compose project
	// but is not a service of the project proper — it exists to read volumes
	// while the real containers are down, and project:clone leaves it behind
	// stopped. Counting it made a freshly cloned project look populated, so
	// `start` woke the helper and reported success while the project had no
	// database at all.
	for _, entry := range parseComposePS(out) {
		if entry.Service != "snapshot" {
			return true
		}
	}
	return false
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

// ensureBindMountSources creates the host directories the generated stack binds
// into containers, before docker gets the chance to.
//
// Docker creates a missing bind-mount source itself, and the daemon runs as
// root — so a directory the project needs appears owned by root, inside the
// user's own project. Two things then break, both quietly: Medusa's download
// step saw a project directory that was no longer empty and skipped cloning the
// starter, and `project:remove` could not delete what it had not created.
//
// Creating them here means they belong to whoever ran madock, which is the
// answer to both. Only paths under the project's own `src` mount are touched,
// and only when missing.
func ensureBindMountSources(composeFile string) {
	data, err := os.ReadFile(composeFile)
	if err != nil {
		return
	}

	runDir := paths.GetRunDirPath()
	// `- ./src/<relative path>:<container path>` is how the templates bind a
	// piece of the project. Named volumes and absolute paths are not ours to
	// create.
	pattern := regexp.MustCompile(`(?m)^\s*-\s*\./src/([^:\s]+):`)
	for _, match := range pattern.FindAllStringSubmatch(string(data), -1) {
		relative := strings.TrimSuffix(match[1], "/")
		if relative == "" || strings.Contains(relative, "..") {
			continue
		}
		target := filepath.Join(runDir, relative)
		if paths.IsFileExist(target) {
			continue
		}
		if err := os.MkdirAll(target, 0o755); err != nil {
			logger.Println("could not create the bind-mount directory "+target+":", err)
		}
	}
}

// ReclaimProjectFiles hands the project directory back to whoever runs madock.
//
// Containers run as root unless something says otherwise, so anything they
// write during a build or a dev server belongs to root on the host too —
// Medusa's `.medusa/client/` is the one that showed up first. The user cannot
// delete it, which is how `project:remove`, a command whose whole promise is to
// leave nothing behind, ended with "permission denied" and a directory still
// there.
//
// Best effort by design: it runs before the containers are taken down so the
// project's own image is available, and a failure is reported rather than
// fatal. Removal has to continue either way — stopping at this point would
// leave the user with both the files and the containers.
func ReclaimProjectFiles(projectName string) {
	usr, err := user.Current()
	if err != nil {
		logger.Println("could not tell who is running madock:", err)
		return
	}

	projectConf := configs2.GetProjectConfig(projectName)
	workdir := projectConf["workdir"]
	if workdir == "" {
		workdir = "/var/www/html"
	}
	service := configs2.ResolveMainService(projectConf, "php")
	container := GetContainerName(projectConf, projectName, service)

	own := "chown -R " + usr.Uid + ":" + usr.Gid + " " + workdir
	if out, execErr := exec.Command("docker", "exec", "-u", "root", container, "sh", "-c", own).CombinedOutput(); execErr == nil {
		return
	} else {
		logger.Println("could not reclaim "+workdir+" through "+container+":", execErr, string(out))
	}

	// The container may already be gone — a project that was stopped, or one
	// whose stack failed to come up. A throwaway container mounting the same
	// directory does the same job and needs nothing of the project.
	runDir := paths.GetRunDirPath()
	fallback := exec.Command("docker", "run", "--rm",
		"-v", runDir+":/madock-target",
		"--entrypoint", "sh", "alpine:3",
		"-c", "chown -R "+usr.Uid+":"+usr.Gid+" /madock-target")
	if out, runErr := fallback.CombinedOutput(); runErr != nil {
		logger.Println("could not reclaim "+runDir+":", runErr, string(out))
	}
}

// prepareHomeDirs makes sure the host directories a project links into exist,
// and returns the home directory holding them.
//
// A project's `composer` and `ssh` entries under aruntime are symlinks to
// ~/.composer and ~/.ssh, and every application container bind-mounts
// ~/.ssh/known_hosts. A machine that has never used ssh — a fresh server, a CI
// runner, a container — has none of that, and the symlink then dangles: Docker
// finds something at the path, cannot make a directory of it, and the stack
// dies with "mkdir .../ssh: file exists", which says nothing about ssh. So
// create them, the way ~/.composer already was.
func prepareHomeDirs() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	if !paths.IsFileExist(home + "/.composer") {
		paths.MakeDirsByPath(home + "/.composer")
	}

	if !paths.IsFileExist(home + "/.ssh") {
		if err = os.Chmod(paths.MakeDirsByPath(home+"/.ssh"), 0700); err != nil {
			return "", err
		}
	}

	// The mount is the file, not the directory. Left absent, Docker creates a
	// *directory* named known_hosts inside the container's .ssh, and ssh then
	// fails on every host key with an error about a directory.
	if !paths.IsFileExist(home + "/.ssh/known_hosts") {
		file, createErr := os.OpenFile(home+"/.ssh/known_hosts", os.O_CREATE|os.O_WRONLY, 0600)
		if createErr != nil {
			return "", createErr
		}
		if err = file.Close(); err != nil {
			return "", err
		}
	}

	return home, nil
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

	composerGlobalDir, err := prepareHomeDirs()
	if err != nil {
		logger.Fatal(err)
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
	ensureBindMountSources(composeFile)
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
