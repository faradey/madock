package pwa

import (
	"github.com/faradey/madock/src/command"
	"github.com/faradey/madock/src/helper/cli"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/docker"
	"github.com/faradey/madock/src/helper/logger"
	"os"
	"os/exec"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"pwa"},
		Handler:  Execute,
		Help:     "Execute PWA command",
		Category: "pwa",
	})
}

func Execute() {
	flag := cli.NormalizeCliCommandWithJoin(os.Args[2:])

	projectName := configs.GetProjectName()
	projectConf := configs.GetCurrentProjectConfig()

	service := "nodejs"
	service, user, workdir := cli.GetEnvForUserServiceWorkdir(service, "www-data", projectConf["workdir"])

	cmd := exec.Command("docker", "exec", "-it", "-u", user, docker.GetContainerName(projectConf, projectName, "nodejs"), "bash", "-c", "cd "+workdir+" && "+flag)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		logger.Fatal(err)
	}
}
