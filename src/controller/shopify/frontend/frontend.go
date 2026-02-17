package frontend

import (
	"github.com/faradey/madock/v3/src/command"
	"github.com/faradey/madock/v3/src/helper/cli"
	"github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/docker"
	"github.com/faradey/madock/v3/src/helper/logger"
	"os"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"shopify:web:frontend", "sy:w:f"},
		Handler:  Execute,
		Help:     "Execute Shopify frontend",
		Category: "shopify",
	})
}

func Execute() {
	flag := cli.NormalizeCliCommandWithJoin(os.Args[2:])
	projectName := configs.GetProjectName()
	projectConf := configs.GetCurrentProjectConfig()
	service, user, workdir := cli.GetEnvForUserServiceWorkdir("php", "www-data", projectConf["workdir"])
	err := docker.ContainerExec(docker.GetContainerName(projectConf, projectName, service), user, true, "bash", "-c", "cd "+workdir+"/web/frontend && "+flag)
	if err != nil {
		logger.Fatal(err)
	}
}
