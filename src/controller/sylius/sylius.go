package sylius

import (
	"os"

	"github.com/faradey/madock/v3/src/command"
	cliHelper "github.com/faradey/madock/v3/src/helper/cli"
	"github.com/faradey/madock/v3/src/helper/cli/fmtc"
	"github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/docker"
	"github.com/faradey/madock/v3/src/helper/logger"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"sylius"},
		Handler:  Execute,
		Help:     "Execute Sylius / Symfony console (bin/console <cmd>)",
		Category: "sylius",
	})
}

func Execute() {
	flag := cliHelper.NormalizeCliCommandWithJoin(os.Args[2:])
	projectName := configs.GetProjectName()
	projectConf := configs.GetCurrentProjectConfig()

	if projectConf["platform"] != "sylius" {
		fmtc.Warning("This command is not supported for " + projectConf["platform"])
		return
	}

	workdir := projectConf["workdir"]
	if workdir == "" {
		workdir = "/var/www/html"
	}

	err := docker.ContainerExec(
		docker.GetContainerName(projectConf, projectName, "php"),
		"www-data",
		true,
		"bash", "-c",
		"cd "+workdir+" && php bin/console "+flag,
	)
	if err != nil {
		logger.Fatal(err)
	}
}
