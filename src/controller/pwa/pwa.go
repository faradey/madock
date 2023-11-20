package pwa

import (
	"github.com/faradey/madock/src/configs"
	cliHelper "github.com/faradey/madock/src/helper"
	"log"
	"os"
	"os/exec"
	"strings"
)

func Execute() {
	flag := cliHelper.NormalizeCliCommandWithJoin(os.Args[2:])

	projectName := configs.GetProjectName()
	projectConf := configs.GetCurrentProjectConfig()

	service := "nodejs"
	service, user, workdir := cliHelper.GetEnvForUserServiceWorkdir(service, "www-data", projectConf["WORKDIR"])

	cmd := exec.Command("docker", "exec", "-it", "-u", user, strings.ToLower(projectConf["CONTAINER_NAME_PREFIX"])+strings.ToLower(projectName)+"-nodejs-1", "bash", "-c", "cd "+workdir+" && "+flag)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}
