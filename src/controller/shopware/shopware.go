package shopware

import (
	"os"

	"github.com/faradey/madock/v3/src/command"
	cliHelper "github.com/faradey/madock/v3/src/helper/cli"
	"github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/docker"
	"github.com/faradey/madock/v3/src/helper/logger"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"shopware", "sw"},
		Handler:  Execute,
		Help:     "Execute Shopware CLI",
		Category: "shopware",
	})
	command.Register(&command.Definition{
		Aliases:  []string{"shopware:bin", "sw:b"},
		Handler:  ExecuteBin,
		Help:     "Execute Shopware bin/console",
		Category: "shopware",
	})
	command.Register(&command.Definition{
		Aliases:  []string{"shopware:consume", "sw:c"},
		Handler:  ExecuteConsume,
		Help:     "Run Shopware messenger consumer (foreground) — for debugging",
		Category: "shopware",
	})
}

func Execute() {
	flag := cliHelper.NormalizeCliCommandWithJoin(os.Args[2:])
	projectName := configs.GetProjectName()
	projectConf := configs.GetCurrentProjectConfig()
	err := docker.ContainerExec(docker.GetContainerName(projectConf, projectName, "php"), "www-data", true, "bash", "-c", "cd "+projectConf["workdir"]+" && php bin/console "+flag)
	if err != nil {
		logger.Fatal(err)
	}
}

func ExecuteBin() {
	flag := cliHelper.NormalizeCliCommandWithJoin(os.Args[2:])
	projectName := configs.GetProjectName()
	projectConf := configs.GetCurrentProjectConfig()
	err := docker.ContainerExec(docker.GetContainerName(projectConf, projectName, "php"), "www-data", true, "bash", "-c", "cd "+projectConf["workdir"]+" && bin/"+flag)
	if err != nil {
		logger.Fatal(err)
	}
}

// ExecuteConsume runs `bin/console messenger:consume` as www-data with sane
// defaults (async receiver, hourly time-limit, verbose). Extra args from the
// command line are appended verbatim — e.g. `madock sw:c failed` to drain the
// failed transport. Use this for foreground debugging; for a long-running
// worker prefer the messenger sidecar service (shopware/messenger/enabled).
func ExecuteConsume() {
	projectName := configs.GetProjectName()
	projectConf := configs.GetCurrentProjectConfig()

	args := "async --time-limit=3600 -vv"
	if len(os.Args) > 2 {
		args = cliHelper.NormalizeCliCommandWithJoin(os.Args[2:])
	}

	err := docker.ContainerExec(
		docker.GetContainerName(projectConf, projectName, "php"),
		"www-data",
		true,
		"bash", "-c",
		"cd "+projectConf["workdir"]+" && php bin/console messenger:consume "+args,
	)
	if err != nil {
		logger.Fatal(err)
	}
}
