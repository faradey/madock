package platform

import (
	"os"
	"os/exec"
	"os/user"
	"strings"

	"github.com/faradey/madock/v4/src/helper/cli/attr"
	"github.com/faradey/madock/v4/src/helper/cli/fmtc"
	"github.com/faradey/madock/v4/src/helper/configs/aruntime/project"
	"github.com/faradey/madock/v4/src/helper/docker"
	"github.com/faradey/madock/v4/src/helper/logger"
	"github.com/faradey/madock/v4/src/helper/paths"
)

// BaseHandler provides default implementation for common operations
type BaseHandler struct {
	MainContainer string
	ChownDirs     []string
	HasCron       bool
}

// GetMainContainer returns the main container name
func (h *BaseHandler) GetMainContainer() string {
	if h.MainContainer != "" {
		return h.MainContainer
	}
	return "php"
}

// GetChownDirs returns directories to chown
func (h *BaseHandler) GetChownDirs(projectConf map[string]string) []string {
	if len(h.ChownDirs) > 0 {
		dirs := make([]string, len(h.ChownDirs))
		for i, dir := range h.ChownDirs {
			if dir == "workdir" {
				dirs[i] = projectConf["workdir"]
			} else {
				dirs[i] = dir
			}
		}
		return dirs
	}
	return []string{projectConf["workdir"]}
}

// SupportsCron returns whether this platform supports cron
func (h *BaseHandler) SupportsCron() bool {
	return h.HasCron
}

// Start starts the containers for a project
func (h *BaseHandler) Start(projectName string, withChown bool, projectConf map[string]string) {
	// Regenerates the compose files and build context from the current config.
	docker.UpNginx(projectName)

	// `docker compose start` only wakes existing containers: it reads the
	// compose file for names, not for content, so a changed image or service
	// definition would be ignored and the project would keep running the old
	// one. When the generated stack no longer matches what the containers were
	// created from, the only honest move is to recreate them.
	if project.NeedsRecreate(projectName) {
		// What changed decides how much is touched. A vhost edit is an nginx
		// reload; a php Dockerfile is one container; a network is everything.
		// Recreating the whole stack for the first of those cost a production
		// store a minute without its database — see project/stackdiff.go.
		if diff := project.DiffStack(projectName); diff.Narrow() {
			fmtc.WarningLn("Configuration changed for " + describeDiff(diff) + " since these containers were created.")
			if err := docker.ApplyStackDiff(projectName, diff, withChown); err != nil {
				fmtc.WarningIconLn(err.Error())
				fmtc.ToDoLn("madock rebuild   # recreates the whole stack")
				return
			}
			// The rest of the stack was left alone and may be stopped; fall
			// through to wake it.
		} else {
			fmtc.WarningLn("Configuration changed since these containers were created.")
			fmtc.ToDoLn("Recreating containers")
			docker.UpProjectWithBuild(projectName, withChown)
			return
		}
	}

	// Nothing to wake. `docker compose start` succeeds with nothing to do when
	// the containers do not exist, so this used to report a started project and
	// leave the machine empty — most visibly right after project:clone, which
	// removes the containers to load the copied data and leaves a configuration
	// fingerprint that still matches.
	if !docker.HasContainers(projectName) {
		fmtc.ToDoLn("Creating containers")
		docker.UpProjectWithBuild(projectName, withChown)
		return
	}

	pp := paths.NewProjectPaths(projectName)
	profilesOn := []string{
		"compose",
		"-f", pp.DockerCompose(),
		"-f", pp.DockerComposeOverride(),
		"start",
	}

	cmd := exec.Command("docker", profilesOn...)
	if !attr.IsQuiet {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	err := cmd.Run()

	if err != nil {
		fmtc.ToDoLn("Creating containers")
		docker.UpProjectWithBuild(projectName, withChown)
	} else {
		// The containers match the rendered stack — the fingerprint said so
		// above — so this is the moment to record what each one was created
		// from. A project that predates the per-service record gets one here,
		// on its first start, instead of a full recreate on its first change.
		project.RecordApplied(projectName)

		if withChown {
			h.executeChown(projectName, projectConf)
		}

		if h.SupportsCron() {
			cronEnabled := false
			if val, ok := projectConf["cron/enabled"]; ok && val == "true" {
				cronEnabled = true
			}
			docker.CronExecute(projectName, cronEnabled, false)
		}
	}
}

// Stop stops the containers for a project
func (h *BaseHandler) Stop(projectName string) {
	pp := paths.NewProjectPaths(projectName)
	profilesOn := []string{
		"compose",
		"-f", pp.DockerCompose(),
		"-f", pp.DockerComposeOverride(),
		"stop",
	}

	cmd := exec.Command("docker", profilesOn...)
	if !attr.IsQuiet {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	err := cmd.Run()
	if err != nil {
		logger.Fatal(err)
	}
}

// executeChown runs chown for the configured directories
func (h *BaseHandler) executeChown(projectName string, projectConf map[string]string) {
	usr, err := user.Current()
	if err != nil {
		logger.Fatal(err)
	}

	// Build chown command for all directories.
	//
	// Each directory is chowned only if it is there. The list is declared per
	// platform, but what exists is decided by the image: a `custom` project with
	// no PHP runs an `app` container with no composer home, and the chain used
	// to stop at the first missing directory and take the whole command with it
	// — `start --with-chown` exited 1 on every such project, after the
	// containers were already up, so the flag was unusable and the failure said
	// nothing about why.
	//
	// A missing directory is skipped; a chown that genuinely cannot run still
	// fails, which is the half worth keeping loud.
	chownCmd := ""
	dirs := h.GetChownDirs(projectConf)
	for i, dir := range dirs {
		if i > 0 {
			chownCmd += " && "
		}
		chownCmd += "if [ -e " + dir + " ]; then chown -R " + usr.Uid + ":" + usr.Gid + " " + dir + "; fi"
	}

	containerName := docker.GetContainerName(projectConf, projectName, ResolveMainService(projectConf, h.GetMainContainer()))
	err = docker.ContainerExec(containerName, "root", true, "bash", "-c", chownCmd)
	if err != nil {
		logger.Fatal(err)
	}
}

// describeDiff names what a narrowed change touches, in the terms the user
// configured: services, not hashes.
func describeDiff(diff project.StackDiff) string {
	var parts []string
	if len(diff.Recreate) > 0 {
		parts = append(parts, strings.Join(diff.Recreate, ", "))
	}
	for _, service := range diff.Restart {
		parts = append(parts, service+" (files)")
	}
	return strings.Join(parts, ", ")
}
