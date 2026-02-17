package shopware

import (
	"os"

	"github.com/faradey/madock/src/command"
	cliHelper "github.com/faradey/madock/src/helper/cli"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/docker"
	"github.com/faradey/madock/src/helper/logger"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"shopware", "sw"},
		Handler:  Execute,
		Help:     "Execute Shopware CLI",
		Category: "shopware",
	})
	command.Register(&command.Definition{
		Aliases:  []string{"shopware:bin", "sw:b"},
		Handler:  ExecuteBin,
		Help:     "Execute Shopware bin/console",
		Category: "shopware",
	})
}

func Execute() {
	flag := cliHelper.NormalizeCliCommandWithJoin(os.Args[2:])
	projectName := configs.GetProjectName()
	projectConf := configs.GetCurrentProjectConfig()
	err := docker.ContainerExec(docker.GetContainerName(projectConf, projectName, "php"), "www-data", true, "bash", "-c", "cd "+projectConf["workdir"]+" && php bin/console "+flag)
	if err != nil {
		logger.Fatal(err)
	}
}

func ExecuteBin() {
	flag := cliHelper.NormalizeCliCommandWithJoin(os.Args[2:])
	projectName := configs.GetProjectName()
	projectConf := configs.GetCurrentProjectConfig()
	err := docker.ContainerExec(docker.GetContainerName(projectConf, projectName, "php"), "www-data", true, "bash", "-c", "cd "+projectConf["workdir"]+" && bin/"+flag)
	if err != nil {
		logger.Fatal(err)
	}
}
