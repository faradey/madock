package docker

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"

	configs2 "github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/logger"
	"github.com/faradey/madock/src/helper/paths"
)

// UpWithBuild starts both nginx proxy and project containers with build
func UpWithBuild(projectName string, withChown bool) {
	UpNginxWithBuild(projectName, true)
	UpProjectWithBuild(projectName, withChown)
}

// Down stops project containers
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
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			fmt.Println(err)
		}
	}
}

// Kill forcefully stops project containers
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
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			fmt.Println(err)
		}
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
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
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
		cmd := exec.Command("docker", "exec", "-it", "-u", "root", GetContainerName(projectConf, projectName, "php"), "bash", "-c", "chown -R "+usr.Uid+":"+usr.Gid+" "+projectConf["workdir"]+" && chown -R "+usr.Uid+":"+usr.Gid+" /var/www/.composer")
		/* for .npm for futures +" && chown -R "+usr.Uid+":"+usr.Gid+" /var/www/.npm" */
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			logger.Fatal(err)
		}
	}
}

// dockerComposePull pulls images for docker-compose
func dockerComposePull(composeFiles []string) {
	composeFiles = append(composeFiles, "pull")
	cmd := exec.Command("docker", composeFiles...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		logger.Fatal(err)
	}
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
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
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
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			fmt.Println(err)
		}
	}
}
