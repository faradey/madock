package shopware

import (
	"os"
	"strings"

	"github.com/faradey/madock/v4/src/command"
	cliHelper "github.com/faradey/madock/v4/src/helper/cli"
	"github.com/faradey/madock/v4/src/helper/cli/fmtc"
	"github.com/faradey/madock/v4/src/helper/configs"
	"github.com/faradey/madock/v4/src/helper/docker"
	"github.com/faradey/madock/v4/src/helper/logger"
)

func init() {
	command.Register(&command.Definition{
		Aliases: []string{"shopware", "sw"},
		Handler: Execute,
		// Not "Execute Shopware CLI". This runs `php bin/console`, and Shopware
		// CLI is a different program — github.com/shopware/shopware-cli, a
		// separate Go binary that validates an extension against the store's
		// requirements and builds its zip. madock does not wrap it at all, so the
		// old wording sent somebody looking for a tool this command is not: they
		// found the name, ran it, and got a Symfony console.
		Help:     "Execute Shopware bin/console",
		Category: "shopware",
		// The arguments are bin/console's.
		PassThrough: true,
	})
	command.Register(&command.Definition{
		Aliases: []string{"shopware:bin", "sw:b"},
		Handler: ExecuteBin,
		// The wider of the two, despite the narrower name: this runs anything in
		// the project's bin/, while `shopware` above is locked to bin/console.
		Help:     "Run an executable from the project's bin/ directory",
		Category: "shopware",
		// The arguments belong to whatever in bin/ is being run.
		PassThrough: true,
	})
	command.Register(&command.Definition{
		Aliases: []string{"shopware:cli", "sw:cli"},
		Handler: ExecuteCli,
		// The vendor's own tooling, and a different program from bin/console: it
		// validates an extension against the store's requirements, builds it and
		// zips it. `sw:c` is already the messenger consumer, hence `sw:cli`.
		Help:     "Execute shopware-cli (extension validate, build, zip). Needs shopware/cli/enabled",
		Category: "shopware",
		// The arguments are shopware-cli's.
		PassThrough: true,
	})
	command.Register(&command.Definition{
		Aliases:  []string{"shopware:consume", "sw:c"},
		Handler:  ExecuteConsume,
		Help:     "Run Shopware messenger consumer (foreground) — for debugging",
		Category: "shopware",
		// The arguments are messenger:consume's.
		PassThrough: true,
	})
}

func Execute() {
	flag := cliHelper.NormalizeCliCommandWithJoin(os.Args[2:])
	projectName := configs.GetProjectName()
	projectConf := configs.GetCurrentProjectConfig()
	err := docker.ContainerExec(docker.GetContainerName(projectConf, projectName, "php"), "www-data", true, "bash", "-c", "cd "+projectConf["workdir"]+" && php bin/console "+flag)
	if err != nil {
		logger.FatalChild(err)
	}
}

// ExecuteCli runs shopware-cli in the project's php container.
//
// The refusal in front of it is the point. shopware-cli is downloaded into the
// image at build time and only when shopware/cli/enabled says so, so on a project
// that has not asked for it the exec fails with "shopware-cli: command not found"
// — a message about a missing binary, which reads as a broken installation rather
// than as a setting nobody turned on. Naming the key and the rebuild costs one
// config read and saves the search.
func ExecuteCli() {
	projectConf := configs.GetCurrentProjectConfig()

	if strings.ToLower(projectConf["shopware/cli/enabled"]) != "true" {
		fmtc.ErrorLn("shopware-cli is not installed in this project's php image.")
		fmtc.ToDoLn("madock config:set --name shopware/cli/enabled --value true")
		fmtc.ToDoLn("madock rebuild")
		os.Exit(1)
	}

	flag := cliHelper.NormalizeCliCommandWithJoin(os.Args[2:])
	projectName := configs.GetProjectName()
	err := docker.ContainerExec(docker.GetContainerName(projectConf, projectName, "php"), "www-data", true, "bash", "-c", "cd "+projectConf["workdir"]+" && shopware-cli "+flag)
	if err != nil {
		logger.FatalChild(err)
	}
}

func ExecuteBin() {
	flag := cliHelper.NormalizeCliCommandWithJoin(os.Args[2:])
	projectName := configs.GetProjectName()
	projectConf := configs.GetCurrentProjectConfig()
	err := docker.ContainerExec(docker.GetContainerName(projectConf, projectName, "php"), "www-data", true, "bash", "-c", "cd "+projectConf["workdir"]+" && bin/"+flag)
	if err != nil {
		logger.FatalChild(err)
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
		logger.FatalChild(err)
	}
}
