package cloud

import (
	"os"
	"strings"

	"github.com/faradey/madock/src/command"
	cliHelper "github.com/faradey/madock/src/helper/cli"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/docker"
	"github.com/faradey/madock/src/helper/logger"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"magento-cloud", "cloud"},
		Handler:  Execute,
		Help:     "Execute Magento Cloud CLI",
		Category: "magento",
	})
}

type ArgsStruct struct {
	attr.ArgumentsWithArgs
}

func Execute() {
	attr.Parse(new(ArgsStruct))

	projectConf := configs.GetCurrentProjectConfig()
	if projectConf["platform"] == "magento2" {
		flag := cliHelper.NormalizeCliCommandWithJoin(os.Args[2:])
		flag = strings.Replace(flag, "$project", projectConf["magento/cloud/project_name"], -1)

		projectName := configs.GetProjectName()
		err := docker.ContainerExec(docker.GetContainerName(projectConf, projectName, "php"), "www-data", true, "bash", "-c", "cd "+projectConf["workdir"]+" && magento-cloud "+flag)
		if err != nil {
			logger.Fatal(err)
		}
	} else {
		fmtc.Warning("This command is not supported for " + projectConf["platform"])
	}
}
