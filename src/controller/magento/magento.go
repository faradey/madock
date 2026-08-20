package magento

import (
	"github.com/faradey/madock/v4/src/command"
	cliHelper "github.com/faradey/madock/v4/src/helper/cli"
	"github.com/faradey/madock/v4/src/helper/configs"
	"github.com/faradey/madock/v4/src/helper/docker"
	"github.com/faradey/madock/v4/src/helper/logger"
	"os"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"magento", "m"},
		Handler:  Execute,
		Help:     "Execute Magento CLI",
		Category: "magento",
		// The arguments are bin/magento's.
		PassThrough: true,
	})
}

func Execute() {
	flag := cliHelper.NormalizeCliCommandWithJoin(os.Args[2:])
	projectName := configs.GetProjectName()
	projectConf := configs.GetCurrentProjectConfig()
	err := docker.ContainerExec(docker.GetContainerName(projectConf, projectName, "php"), "www-data", true, "bash", "-c", "cd "+projectConf["workdir"]+" && php bin/magento "+flag)
	if err != nil {
		logger.Fatal(err)
	}
}
