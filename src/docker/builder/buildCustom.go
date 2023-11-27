package builder

import (
	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/controller/general/cron"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	"github.com/faradey/madock/src/helper/paths"
	"log"
	"os"
	"os/exec"
	"os/user"
	"strings"
)

func StartCustom(withChown bool, projectConf map[string]string) {
	projectName := configs.GetProjectName()
	UpNginx()
	composeFileOS := paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/docker-compose.override.yml"
	profilesOn := []string{
		"compose",
		"-f",
		paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/docker-compose.yml",
		"-f",
		composeFileOS,
		"--profile",
		"elasticsearchtrue",
		"--profile",
		"opensearchtrue",
		"--profile",
		"redisdbtrue",
		"--profile",
		"rabbitmqtrue",
		"--profile",
		"kibanatrue",
		"--profile",
		"opensearchdashboardtrue",
		"--profile",
		"phpmyadmintrue",
		"--profile",
		"db2true",
		"--profile",
		"phpmyadmin2true",
		"--profile",
		"xdebugtrue",
		"start",
	}
	cmd := exec.Command("docker", profilesOn...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmtc.ToDoLn("Creating containers")
		upProjectWithBuild(withChown)
	} else {
		if withChown {
			projectName := configs.GetProjectName()
			usr, _ := user.Current()
			cmd := exec.Command("docker", "exec", "-it", "-u", "root", strings.ToLower(projectConf["CONTAINER_NAME_PREFIX"])+strings.ToLower(projectName)+"-php-1", "bash", "-c", "chown -R "+usr.Uid+":"+usr.Gid+" "+projectConf["WORKDIR"]+" && chown -R "+usr.Uid+":"+usr.Gid+" /var/www/.composer")
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err = cmd.Run()
			if err != nil {
				log.Fatal(err)
			}
		}

		if val, ok := projectConf["CRON_ENABLED"]; ok && val == "true" {
			cron.Execute(true, false)
		} else {
			cron.Execute(false, false)
		}
	}
}

func StopCustom() {
	projectName := configs.GetProjectName()
	composeFileOS := paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/docker-compose.override.yml"
	profilesOn := []string{
		"compose",
		"-f",
		paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/docker-compose.yml",
		"-f",
		composeFileOS,
		"--profile",
		"elasticsearchtrue",
		"--profile",
		"opensearchtrue",
		"--profile",
		"redisdbtrue",
		"--profile",
		"rabbitmqtrue",
		"--profile",
		"kibanatrue",
		"--profile",
		"opensearchdashboardtrue",
		"--profile",
		"phpmyadmintrue",
		"--profile",
		"db2true",
		"--profile",
		"phpmyadmin2true",
		"--profile",
		"xdebugtrue",
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
