package cli

import (
	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/helper/cli"
	"log"
	"os"
	"os/exec"
	"strings"
)

func Cli() {
	flag := cli.NormalizeCliCommandWithJoin(os.Args[2:])
	projectName := configs.GetProjectName()
	projectConf := configs.GetCurrentProjectConfig()
	service := "php"
	if projectConf["PLATFORM"] == "pwa" {
		service = "nodejs"
	}

	service, user, workdir := cli.GetEnvForUserServiceWorkdir(service, "www-data", "")

	if workdir != "" {
		workdir = "cd " + workdir + " && "
	}

	cmd := exec.Command("docker", "exec", "-it", "-u", user, strings.ToLower(projectConf["CONTAINER_NAME_PREFIX"])+strings.ToLower(projectName)+"-"+service+"-1", "bash", "-c", workdir+flag)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}
