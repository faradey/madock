package builder

import (
	"github.com/faradey/madock/src/cli/attr"
	"github.com/faradey/madock/src/cli/fmtc"
	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/helper/paths"
	"log"
	"os"
	"os/exec"
	"os/user"
	"strings"
)

func StartPWA(withChown bool) {
	projectName := configs.GetProjectName()
	UpNginx()
	composeFileOS := paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/docker-compose.override.yml"
	profilesOn := []string{
		"compose",
		"-f",
		paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/docker-compose.yml",
		"-f",
		composeFileOS,
		"start",
	}
	cmd := exec.Command("docker", profilesOn...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmtc.ToDoLn("Creating containers")
		upProjectWithBuild(attr.Options.WithChown)
	} else if withChown {
		projectName := configs.GetProjectName()
		projectConf := configs.GetCurrentProjectConfig()
		usr, _ := user.Current()
		cmd := exec.Command("docker", "exec", "-it", "-u", "root", strings.ToLower(projectConf["CONTAINER_NAME_PREFIX"])+strings.ToLower(projectName)+"-nodejs-1", "bash", "-c", "chown -R "+usr.Uid+":"+usr.Gid+" "+projectConf["WORKDIR"])
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			log.Fatal(err)
		}
	}
}

func StopPWA() {
	projectName := configs.GetProjectName()
	composeFileOS := paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/docker-compose.override.yml"
	profilesOn := []string{
		"compose",
		"-f",
		paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/docker-compose.yml",
		"-f",
		composeFileOS,
		"stop",
	}
	cmd := exec.Command("docker", profilesOn...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}
