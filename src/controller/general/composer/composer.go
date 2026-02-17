package composer

import (
	"os"

	"github.com/faradey/madock/src/command"
	"github.com/faradey/madock/src/helper/cli"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/docker"
	"github.com/faradey/madock/src/helper/logger"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"composer"},
		Handler:  Execute,
		Help:     "Execute composer command",
		Category: "general",
	})
}

func Execute() {
	flag := cli.NormalizeCliCommandWithJoin(os.Args[2:])

	projectName := configs.GetProjectName()
	projectConf := configs.GetCurrentProjectConfig()

	service, user, workdir := cli.GetEnvForUserServiceWorkdir("php", "www-data", projectConf["workdir"])

	workdir += "/" + projectConf["composer_dir"]

	err := docker.ContainerExec(docker.GetContainerName(projectConf, projectName, service), user, true, "bash", "-c", "cd "+workdir+" && composer "+flag)
	if err != nil {
		logger.Fatal(err)
	}
}
