package magento

import (
	"github.com/faradey/madock/src/helper/cli"
	"github.com/faradey/madock/src/helper/cli/arg_struct"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/docker"
	"github.com/faradey/madock/src/helper/logger"
	"os"
	"os/exec"
)

func Execute() {
	args := attr.Parse(new(arg_struct.ControllerGeneralMagentoDiff)).(*arg_struct.ControllerGeneralMagentoDiff)

	oldPath := args.Old
	newPath := args.New

	if oldPath == "" || newPath == "" {
		logger.Fatal("The --old and --new options are required.")
	}

	projectName := configs.GetProjectName()
	projectConf := configs.GetCurrentProjectConfig()
	service, user, workdir := cli.GetEnvForUserServiceWorkdir("php", "www-data", projectConf["workdir"])

	publicDir := args.Path
	cmdArgs := []string{"exec", "-it", "-u", user, docker.GetContainerName(projectConf, projectName, service), "php", "/var/www/scripts/php/magento-diff.php", workdir, oldPath, newPath, publicDir}

	cmd := exec.Command("docker", cmdArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		logger.Fatal(err)
	}
}
