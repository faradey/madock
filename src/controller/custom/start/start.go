package start

import (
	"github.com/faradey/madock/src/helper/cli/fmtc"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/docker"
	"github.com/faradey/madock/src/helper/logger"
	"github.com/faradey/madock/src/helper/paths"
	"os"
	"os/exec"
	"os/user"
)

func Execute(withChown bool, projectConf map[string]string) {
	projectName := configs.GetProjectName()
	docker.UpNginx(projectName)
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
		docker.UpProjectWithBuild(projectName, withChown)
	} else {
		if withChown {
			usr, _ := user.Current()
			cmd = exec.Command("docker", "exec", "-it", "-u", "root", docker.GetContainerName(projectConf, projectName, "php"), "bash", "-c", "chown -R "+usr.Uid+":"+usr.Gid+" "+projectConf["workdir"]+" && chown -R "+usr.Uid+":"+usr.Gid+" /var/www/.composer")
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err = cmd.Run()
			if err != nil {
				logger.Fatal(err)
			}
		}

		if val, ok := projectConf["cron/enabled"]; ok && val == "true" {
			docker.CronExecute(true, false)
		} else {
			docker.CronExecute(false, false)
		}
	}
}
