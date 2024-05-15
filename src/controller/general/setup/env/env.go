package env

import (
	"encoding/json"
	"github.com/faradey/madock/src/helper/cli"
	"github.com/faradey/madock/src/helper/cli/arg_struct"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/docker"
	"github.com/faradey/madock/src/helper/logger"
	"github.com/faradey/madock/src/helper/paths"
	"os"
	"os/exec"
)

func Execute() {
	args := attr.Parse(new(arg_struct.ControllerGeneralSetupEnv)).(*arg_struct.ControllerGeneralSetupEnv)

	envFile := paths.GetRunDirPath() + "/app/etc/env.php"
	if paths.IsFileExist(envFile) && !args.Force {
		logger.Fatal("The env.php file is already exist.")
	} else {
		data, err := json.Marshal(configs.GetCurrentProjectConfig())
		if err != nil {
			logger.Fatal(err)
		}

		conf := string(data)
		host := args.Host
		projectName := configs.GetProjectName()
		projectConf := configs.GetCurrentProjectConfig()
		service, user, _ := cli.GetEnvForUserServiceWorkdir("php", "www-data", "")
		cmd := exec.Command("docker", "exec", "-it", "-u", user, docker.GetContainerName(projectConf, projectName, service), "php", "/var/www/scripts/php/env-create.php", conf, host)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			logger.Fatal(err)
		}
	}
}
