package prune

import (
	"github.com/faradey/madock/src/controller/general/proxy"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/docker"
	"github.com/faradey/madock/src/helper/paths"
)

type ArgsStruct struct {
	attr.Arguments
	WithVolumes bool `arg:"-v,--with-volumes" help:"With Volumes"`
}

func Execute() {
	args := attr.Parse(new(ArgsStruct)).(*ArgsStruct)

	if configs.IsHasConfig("") {
		docker.Down(args.WithVolumes)
		if len(paths.GetActiveProjects()) == 0 {
			proxy.Execute("prune")
		}
		fmtc.SuccessLn("Done")
	} else {
		fmtc.WarningLn("Set up the project")
		fmtc.ToDoLn("Run madock setup")
	}
}
