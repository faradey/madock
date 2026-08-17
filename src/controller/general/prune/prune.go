package prune

import (
	"os"

	"github.com/faradey/madock/v3/src/command"
	"github.com/faradey/madock/v3/src/controller/general/proxy"
	"github.com/faradey/madock/v3/src/helper/cli/arg_struct"
	"github.com/faradey/madock/v3/src/helper/cli/attr"
	"github.com/faradey/madock/v3/src/helper/cli/fmtc"
	"github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/docker"
	"github.com/faradey/madock/v3/src/helper/paths"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"prune"},
		Handler:  Execute,
		Help:     "Prune Docker resources",
		Category: "general",
		ArgsType: new(arg_struct.ControllerGeneralPrune),
	})
}

func Execute() {
	args := attr.Parse(new(arg_struct.ControllerGeneralPrune)).(*arg_struct.ControllerGeneralPrune)

	// The name promises `docker system prune` and the body is `docker compose
	// down` for the current project — with --with-volumes, its data volumes and
	// images as well. Whatever it is called, it destroys, so the installation's
	// answer applies here too.
	if !configs.AllowsDestructiveCommands() {
		for _, line := range configs.DestructiveRefusal("prune") {
			fmtc.ErrorLn(line)
		}
		os.Exit(1)
	}

	if configs.IsHasConfig("") {
		projectname := configs.GetProjectName()
		docker.Down(projectname, args.WithVolumes)
		if len(paths.GetActiveProjects()) == 0 {
			proxy.Execute("prune")
		}
		fmtc.SuccessLn("Done")
	} else {
		fmtc.WarningLn("Set up the project")
		fmtc.ToDoLn("Run madock setup")
	}
}
