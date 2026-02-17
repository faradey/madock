package info

import (
	"github.com/faradey/madock/src/command"
	"github.com/faradey/madock/src/controller/platform"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/docker"
	"github.com/faradey/madock/src/helper/logger"
)

type ArgsStruct struct {
	attr.Arguments
}

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"info"},
		Handler:  Info,
		Help:     "Show project info",
		Category: "general",
	})
}

func Info() {
	attr.Parse(new(ArgsStruct))

	projectConf := configs.GetCurrentProjectConfig()
	service := platform.GetMainService(projectConf)

	if projectConf["platform"] == "magento2" {
		projectName := configs.GetProjectName()
		err := docker.ContainerExec(docker.GetContainerName(projectConf, projectName, service), "", true, "php", "/var/www/scripts/php/magento-info.php", projectConf["workdir"])
		if err != nil {
			logger.Fatal(err)
		}
	} else {
		fmtc.Warning("This command is not supported for " + projectConf["platform"])
	}
}
