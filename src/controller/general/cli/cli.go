package cli

import (
	"os"

	"github.com/faradey/madock/src/command"
	"github.com/faradey/madock/src/controller/platform"
	"github.com/faradey/madock/src/helper/cli"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/docker"
	"github.com/faradey/madock/src/helper/logger"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"cli"},
		Handler:  Execute,
		Help:     "Execute CLI in container",
		Category: "general",
	})
}

func Execute() {
	flag := cli.NormalizeCliCommandWithJoin(os.Args[2:])
	projectName := configs.GetProjectName()
	projectConf := configs.GetCurrentProjectConfig()
	service := platform.GetMainService(projectConf)

	service, user, workdir := cli.GetEnvForUserServiceWorkdir(service, "www-data", "")

	interactive := os.Getenv("MADOCK_TTY_ENABLED") != "0"

	if workdir != "" {
		workdir = "cd " + workdir + " && "
	}

	err := docker.ContainerExec(docker.GetContainerName(projectConf, projectName, service), user, interactive, "bash", "-c", workdir+flag)
	if err != nil {
		logger.Fatal(err)
	}
}
