package opencode

import (
	"os"

	"github.com/faradey/madock/v4/src/command"
	"github.com/faradey/madock/v4/src/helper/cli"
	"github.com/faradey/madock/v4/src/helper/configs"
	"github.com/faradey/madock/v4/src/helper/docker"
	"github.com/faradey/madock/v4/src/helper/logger"
	"github.com/faradey/madock/v4/src/helper/opencode"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"opencode"},
		Handler:  Execute,
		Help:     "Execute opencode in a sandboxed container (needs opencode/enabled)",
		Category: "general",
		// The arguments are the opencode CLI's.
		PassThrough: true,
	})
}

func Execute() {
	flag := cli.NormalizeCliCommandWithJoin(os.Args[2:])
	projectName := configs.GetProjectName()
	projectConf := configs.GetCurrentProjectConfig()
	if !opencode.Enabled(projectConf) {
		logger.Fatalln("opencode is off for this project: madock config:set -n opencode/enabled -v true, then madock rebuild")
	}

	service, user, workdir := cli.GetEnvForUserServiceWorkdir(opencode.Service, "www-data", projectConf["workdir"])

	if workdir != "" {
		workdir = "cd " + workdir + " && "
	}

	err := docker.ContainerExec(docker.GetContainerName(projectConf, projectName, service), user, true, "bash", "-c", workdir+"opencode "+flag)
	if err != nil {
		logger.FatalChild(err)
	}
}
