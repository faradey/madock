package logs

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/faradey/madock/v4/src/command"
	"github.com/faradey/madock/v4/src/controller/platform"
	"github.com/faradey/madock/v4/src/helper/cli/arg_struct"
	"github.com/faradey/madock/v4/src/helper/cli/attr"
	"github.com/faradey/madock/v4/src/helper/cli/fmtc"
	"github.com/faradey/madock/v4/src/helper/configs"
	"github.com/faradey/madock/v4/src/helper/docker"
	"github.com/faradey/madock/v4/src/helper/logger"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"logs"},
		Handler:  Execute,
		Help:     "Show container logs",
		Category: "general",
		ArgsType: new(arg_struct.ControllerGeneralLogs),
	})
}

// chooseService settles which container's log to show.
//
// Two ways of naming the same thing, so the only case worth a word is naming it
// twice differently: showing one of them would be a guess, and a guess here
// reads as logs — the reader concludes the other service is quiet.
func chooseService(positional, flag, fallback string) (string, error) {
	if positional != "" && flag != "" && positional != flag {
		return "", fmt.Errorf("two services asked for at once: %q and --service %q. Name one", positional, flag)
	}

	if flag != "" {
		return flag, nil
	}
	if positional != "" {
		return positional, nil
	}

	return fallback, nil
}

func Execute() {
	args := attr.Parse(new(arg_struct.ControllerGeneralLogs)).(*arg_struct.ControllerGeneralLogs)

	projectConf := configs.GetCurrentProjectConfig()

	service, err := chooseService(args.Name, args.Service, platform.GetMainService(projectConf))
	if err != nil {
		fmtc.ErrorLn(err.Error())
		os.Exit(1)
	}

	projectName := configs.GetProjectName()
	cmd := exec.Command("docker", "logs", docker.GetContainerName(projectConf, projectName, service))
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logger.Fatal(err)
	}
}
