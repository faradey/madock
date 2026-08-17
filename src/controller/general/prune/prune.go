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

	// Only the flag is destructive, and the difference is the whole of it.
	//
	// Plain `prune` is `docker compose down`: containers and the network go,
	// the volumes stay, the images stay, the project directory and its registry
	// entry are not touched. `madock start` puts it back. That is `stop` with
	// the containers removed, and guarding it would be both an obstacle and
	// inconsistent — `stop` takes the same site down and nothing stands in
	// front of it.
	//
	// `--with-volumes` is `down -v --rmi all`: the data volumes and the images.
	// The database is gone and nothing brings it back, which is what the
	// installation's answer is about.
	if args.WithVolumes && !configs.AllowsDestructiveCommands() {
		for _, line := range configs.DestructiveRefusal("prune --with-volumes") {
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
