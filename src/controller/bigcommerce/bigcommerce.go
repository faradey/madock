package bigcommerce

import (
	"os"

	"github.com/faradey/madock/v3/src/command"
	cliHelper "github.com/faradey/madock/v3/src/helper/cli"
	"github.com/faradey/madock/v3/src/helper/cli/fmtc"
	"github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/docker"
	"github.com/faradey/madock/v3/src/helper/logger"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"bigcommerce", "bc"},
		Handler:  Execute,
		Help:     "Run a command inside the BigCommerce project's main container (preset-aware)",
		Category: "bigcommerce",
	})
}

func Execute() {
	flag := cliHelper.NormalizeCliCommandWithJoin(os.Args[2:])
	projectName := configs.GetProjectName()
	projectConf := configs.GetCurrentProjectConfig()

	if projectConf["platform"] != "bigcommerce" {
		fmtc.Warning("This command is not supported for " + projectConf["platform"])
		return
	}

	workdir := projectConf["workdir"]
	if workdir == "" {
		workdir = "/var/www/html"
	}

	// Pick the container + user based on the preset.
	service := "nodejs"
	user := "node"
	if projectConf["bigcommerce/preset"] == "api-php" {
		service = "php"
		user = "www-data"
	}

	err := docker.ContainerExec(
		docker.GetContainerName(projectConf, projectName, service),
		user,
		true,
		"bash", "-c",
		"cd "+workdir+" && "+flag,
	)
	if err != nil {
		logger.Fatal(err)
	}
}
