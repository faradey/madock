package cli

import (
	"os"
	"strings"

	"github.com/faradey/madock/v4/src/command"
	"github.com/faradey/madock/v4/src/controller/platform"
	"github.com/faradey/madock/v4/src/helper/cli"
	"github.com/faradey/madock/v4/src/helper/cli/fmtc"
	"github.com/faradey/madock/v4/src/helper/configs"
	"github.com/faradey/madock/v4/src/helper/docker"
	"github.com/faradey/madock/v4/src/helper/logger"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"cli"},
		Handler:  Execute,
		Help:     "Execute CLI in container. Optional first argument: --service <name>",
		Category: "general",
		// Everything after the optional --service belongs to the shell.
		PassThrough: true,
	})
}

// takeServiceFlag pulls a leading --service/-s off the argument list and returns
// the rest untouched.
//
// Parsed by hand, and only in first position, because everything after it is a
// command for the container and has flags of its own: a generic parser would try
// to own `-c`, `--force`, `-v` and refuse to pass them on. That is exactly what
// `madock bash --service php -c '…'` runs into — bash does take --service, but its
// parser rejects the command, so until now there was no way to run something in a
// service other than the main one except by setting MADOCK_SERVICE_NAME.
func takeServiceFlag(args []string) (string, []string) {
	if len(args) == 0 {
		return "", args
	}

	first := args[0]
	for _, prefix := range []string{"--service=", "-s="} {
		if strings.HasPrefix(first, prefix) {
			return strings.TrimPrefix(first, prefix), args[1:]
		}
	}
	if first == "--service" || first == "-s" {
		if len(args) < 2 {
			fmtc.ErrorLn("--service needs a service name, for example: madock cli --service php php -v")
			os.Exit(1)
		}
		return args[1], args[2:]
	}

	return "", args
}

func Execute() {
	service, rest := takeServiceFlag(os.Args[2:])
	flag := cli.NormalizeCliCommandWithJoin(rest)
	projectName := configs.GetProjectName()
	projectConf := configs.GetCurrentProjectConfig()
	mainService := platform.GetMainService(projectConf)

	mainService, user, workdir := cli.GetEnvForUserServiceWorkdir(mainService, "www-data", "")
	// An explicit flag beats the ambient MADOCK_SERVICE_NAME: the environment is
	// how a wrapper sets a default, the flag is what the person just typed.
	if service == "" {
		service = mainService
	}

	if workdir != "" {
		workdir = "cd " + workdir + " && "
	}

	err := docker.ContainerExec(docker.GetContainerName(projectConf, projectName, service), user, true, "bash", "-c", workdir+flag)
	if err != nil {
		logger.FatalChild(err)
	}
}
