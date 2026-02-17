package env

import (
	"encoding/json"

	"github.com/faradey/madock/v3/src/command"
	"github.com/faradey/madock/v3/src/helper/cli"
	"github.com/faradey/madock/v3/src/helper/cli/arg_struct"
	"github.com/faradey/madock/v3/src/helper/cli/attr"
	"github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/docker"
	"github.com/faradey/madock/v3/src/helper/logger"
	"github.com/faradey/madock/v3/src/helper/paths"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"setup:env"},
		Handler:  Execute,
		Help:     "Setup environment",
		Category: "setup",
	})
}

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
		err = docker.ContainerExec(docker.GetContainerName(projectConf, projectName, service), user, true, "php", "/var/www/scripts/php/env-create.php", conf, host)
		if err != nil {
			logger.Fatal(err)
		}
	}
}
