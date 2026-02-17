package bash

import (
	"strings"

	"github.com/faradey/madock/v3/src/command"
	"github.com/faradey/madock/v3/src/controller/platform"
	"github.com/faradey/madock/v3/src/helper/cli/arg_struct"
	"github.com/faradey/madock/v3/src/helper/cli/attr"
	"github.com/faradey/madock/v3/src/helper/cli/fmtc"
	"github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/docker"
	"github.com/faradey/madock/v3/src/helper/logger"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"bash"},
		Handler:  Execute,
		Help:     "Execute bash in container",
		Category: "general",
	})
}

var allowedShells = map[string]bool{
	"bash": true,
	"sh":   true,
	"zsh":  true,
	"ash":  true,
	"fish": true,
}

func Execute() {
	args := attr.Parse(new(arg_struct.ControllerGeneralBash)).(*arg_struct.ControllerGeneralBash)

	projectConf := configs.GetCurrentProjectConfig()
	service := platform.GetMainService(projectConf)
	user := "root"

	if args.Service != "" {
		service = args.Service
	}

	if args.User != "" {
		user = args.User
	}

	projectName := configs.GetProjectName()
	shell := "bash"
	if args.Shell != "" {
		shell = strings.TrimSpace(args.Shell)
		if !allowedShells[shell] {
			fmtc.ErrorLn("Invalid shell. Allowed shells: bash, sh, zsh, ash, fish")
			return
		}
	}
	err := docker.ContainerExec(docker.GetContainerName(projectConf, projectName, service), user, true, shell)
	if err != nil {
		logger.Fatal(err)
	}
}
