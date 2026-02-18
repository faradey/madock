package diff

import (
	"os"
	"os/exec"
	"strings"

	"github.com/faradey/madock/v3/src/command"
	"github.com/faradey/madock/v3/src/helper/cli"
	"github.com/faradey/madock/v3/src/helper/cli/arg_struct"
	"github.com/faradey/madock/v3/src/helper/cli/attr"
	"github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/docker"
	"github.com/faradey/madock/v3/src/helper/logger"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"diff"},
		Handler:  Execute,
		Help:     "Show diff",
		Category: "general",
	})
}

func Execute() {
	args := attr.Parse(new(arg_struct.ControllerGeneralDiff)).(*arg_struct.ControllerGeneralDiff)

	platform := strings.ToLower(args.Platform)
	if platform == "" {
		logger.Fatal("The --platform option is required.")
	}

	oldPath := args.Old
	newPath := args.New

	if oldPath == "" || newPath == "" {
		logger.Fatal("The --old and --new options are required.")
	}

	switch platform {
	case "magento":
		runMagentoDiff(args)
	default:
		logger.Fatal("Unsupported platform: " + platform)
	}
}

func runMagentoDiff(args *arg_struct.ControllerGeneralDiff) {
	projectName := configs.GetProjectName()
	projectConf := configs.GetCurrentProjectConfig()
	service, user, workdir := cli.GetEnvForUserServiceWorkdir("php", "www-data", projectConf["workdir"])

	ttyFlag := "-i"
	if docker.IsTTYAvailable() {
		ttyFlag = "-it"
	}
	cmdArgs := []string{"exec", ttyFlag, "-u", user, docker.GetContainerName(projectConf, projectName, service), "php", "/var/www/scripts/php/diff.php", workdir, args.Old, args.New, args.Path, args.Platform}

	cmd := exec.Command("docker", cmdArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logger.Fatal(err)
	}
}
