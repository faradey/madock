package logs

import (
	"os"
	"os/exec"

	"github.com/faradey/madock/src/command"
	"github.com/faradey/madock/src/controller/platform"
	"github.com/faradey/madock/src/helper/cli/arg_struct"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/docker"
	"github.com/faradey/madock/src/helper/logger"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"logs"},
		Handler:  Execute,
		Help:     "Show container logs",
		Category: "general",
	})
}

func Execute() {
	args := attr.Parse(new(arg_struct.ControllerGeneralLogs)).(*arg_struct.ControllerGeneralLogs)

	projectConf := configs.GetCurrentProjectConfig()
	service := platform.GetMainService(projectConf)

	if args.Service != "" {
		service = args.Service
	}

	projectName := configs.GetProjectName()
	cmd := exec.Command("docker", "logs", docker.GetContainerName(projectConf, projectName, service))
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		logger.Fatal(err)
	}
}
