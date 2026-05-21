package medusa

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
		Aliases:  []string{"medusa"},
		Handler:  Execute,
		Help:     "Execute Medusa CLI",
		Category: "medusa",
	})
}

func Execute() {
	flag := cliHelper.NormalizeCliCommandWithJoin(os.Args[2:])
	projectName := configs.GetProjectName()
	projectConf := configs.GetCurrentProjectConfig()

	if projectConf["platform"] != "medusa" {
		fmtc.Warning("This command is not supported for " + projectConf["platform"])
		return
	}

	err := docker.ContainerExec(
		docker.GetContainerName(projectConf, projectName, "nodejs"),
		"node",
		true,
		"bash", "-c",
		"cd "+projectConf["workdir"]+" && npx medusa "+flag,
	)
	if err != nil {
		logger.Fatal(err)
	}
}
