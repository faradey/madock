package rebuild

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
	Force     bool `arg:"-f,--force" help:"Force"`
	WithChown bool `arg:"-c,--with-chown" help:"With Chown"`
}

func Execute() {
	args := attr.Parse(new(ArgsStruct)).(*ArgsStruct)

	if !configs.IsHasNotConfig() {
		fmtc.SuccessLn("Stop containers")
		if args.Force {
			docker.Kill()
		} else {
			docker.Down(false)
		}
		if len(paths.GetActiveProjects()) == 0 {
			proxy.Execute("stop")
		}
		fmtc.SuccessLn("Start containers in detached mode")
		docker.UpWithBuild(args.WithChown)
		fmtc.SuccessLn("Done")
	} else {
		fmtc.WarningLn("Set up the project")
		fmtc.ToDoLn("Run madock setup")
	}
}
