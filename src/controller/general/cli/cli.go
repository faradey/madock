package cli

import (
	"os"

	"github.com/faradey/madock/v3/src/command"
	"github.com/faradey/madock/v3/src/controller/platform"
	"github.com/faradey/madock/v3/src/helper/cli"
	"github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/docker"
	"github.com/faradey/madock/v3/src/helper/logger"
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

	if workdir != "" {
		workdir = "cd " + workdir + " && "
	}

	err := docker.ContainerExec(docker.GetContainerName(projectConf, projectName, service), user, true, "bash", "-c", workdir+flag)
	if err != nil {
		logger.Fatal(err)
	}
}
