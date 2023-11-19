package composer

import (
	"github.com/faradey/madock/src/configs"
	cliHelper "github.com/faradey/madock/src/helper"
	"log"
	"os"
	"os/exec"
	"strings"
)

func Composer() {
	flag := cliHelper.NormalizeCliCommandWithJoin(os.Args[2:])

	projectName := configs.GetProjectName()
	projectConf := configs.GetCurrentProjectConfig()

	service, user, workdir := cliHelper.GetEnvForUserServiceWorkdir("php", "www-data", projectConf["WORKDIR"])

	cmd := exec.Command("docker", "exec", "-it", "-u", user, strings.ToLower(projectConf["CONTAINER_NAME_PREFIX"])+strings.ToLower(projectName)+"-"+service+"-1", "bash", "-c", "cd "+workdir+" && composer "+flag)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}
