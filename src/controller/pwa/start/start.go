package start

import (
	"github.com/faradey/madock/src/helper/cli/fmtc"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/docker"
	"github.com/faradey/madock/src/helper/paths"
	"log"
	"os"
	"os/exec"
	"os/user"
	"strings"
)

func Execute(withChown bool) {
	projectName := configs.GetProjectName()
	docker.UpNginx()
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
		docker.UpProjectWithBuild(withChown)
	} else if withChown {
		projectConf := configs.GetCurrentProjectConfig()
		usr, _ := user.Current()
		cmd = exec.Command("docker", "exec", "-it", "-u", "root", strings.ToLower(projectConf["CONTAINER_NAME_PREFIX"])+strings.ToLower(projectName)+"-nodejs-1", "bash", "-c", "chown -R "+usr.Uid+":"+usr.Gid+" "+projectConf["WORKDIR"])
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			log.Fatal(err)
		}
	}
}
