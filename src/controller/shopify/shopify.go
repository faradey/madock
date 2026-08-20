package shopify

import (
	"github.com/faradey/madock/v4/src/command"
	"github.com/faradey/madock/v4/src/helper/cli"
	"github.com/faradey/madock/v4/src/helper/configs"
	"github.com/faradey/madock/v4/src/helper/docker"
	"github.com/faradey/madock/v4/src/helper/logger"
	"os"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"shopify", "sy"},
		Handler:  Execute,
		Help:     "Execute Shopify CLI",
		Category: "shopify",
		// The arguments are the Shopify CLI's.
		PassThrough: true,
	})
}

func Execute() {
	flag := cli.NormalizeCliCommandWithJoin(os.Args[2:])
	projectName := configs.GetProjectName()
	projectConf := configs.GetCurrentProjectConfig()
	service, user, workdir := cli.GetEnvForUserServiceWorkdir("php", "www-data", projectConf["workdir"])
	err := docker.ContainerExec(docker.GetContainerName(projectConf, projectName, service), user, true, "bash", "-c", "cd "+workdir+" && "+flag)
	if err != nil {
		logger.Fatal(err)
	}
}
