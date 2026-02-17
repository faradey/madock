package prestashop

import (
	"github.com/faradey/madock/src/command"
	cliHelper "github.com/faradey/madock/src/helper/cli"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/docker"
	"github.com/faradey/madock/src/helper/logger"
	"os"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"prestashop", "ps"},
		Handler:  Execute,
		Help:     "Execute PrestaShop CLI",
		Category: "prestashop",
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
