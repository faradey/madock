package patch

import (
	"os"
	"os/exec"

	"github.com/faradey/madock/v3/src/command"
	"github.com/faradey/madock/v3/src/helper/cli"
	"github.com/faradey/madock/v3/src/helper/cli/arg_struct"
	"github.com/faradey/madock/v3/src/helper/cli/attr"
	"github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/docker"
	"github.com/faradey/madock/v3/src/helper/logger"
	"golang.org/x/term"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"patch:create"},
		Handler:  Execute,
		Help:     "Create patch file",
		Category: "general",
	})
}

func Execute() {
	args := attr.Parse(new(arg_struct.ControllerGeneralPatch)).(*arg_struct.ControllerGeneralPatch)

	filePath := args.File
	patchName := args.Name
	title := args.Title
	force := args.Force

	if filePath == "" {
		logger.Fatal("The --file option is incorrect or not specified.")
	}

	projectName := configs.GetProjectName()
	projectConf := configs.GetCurrentProjectConfig()
	isForce := ""
	if force {
		isForce = "f"
	}
	service, user, workdir := cli.GetEnvForUserServiceWorkdir("php", "www-data", projectConf["workdir"])
	dockerArgs := []string{"exec", "-i"}
	if term.IsTerminal(int(os.Stdin.Fd())) {
		dockerArgs = []string{"exec", "-it"}
	}
	dockerArgs = append(dockerArgs, "-u", user, docker.GetContainerName(projectConf, projectName, service), "php", "/var/www/scripts/php/patch-create.php", workdir, filePath, patchName, title, isForce)
	cmd := exec.Command("docker", dockerArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		logger.Fatal(err)
	}
}
