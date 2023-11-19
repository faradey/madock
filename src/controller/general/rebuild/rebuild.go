package rebuild

import (
	"github.com/faradey/madock/src/cli/attr"
	"github.com/faradey/madock/src/cli/fmtc"
	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/controller/general/proxy"
	"github.com/faradey/madock/src/docker/builder"
	"github.com/faradey/madock/src/paths"
	"github.com/jessevdk/go-flags"
	"log"
	"os"
)

type ArgsStruct struct {
	attr.Arguments
}

func Execute() {
	getArgs()

	if !configs.IsHasNotConfig() {
		fmtc.SuccessLn("Stop containers")
		builder.Down(false)
		if len(paths.GetActiveProjects()) == 0 {
			proxy.Execute("stop")
		}
		fmtc.SuccessLn("Start containers in detached mode")
		builder.UpWithBuild()
		fmtc.SuccessLn("Done")
	} else {
		fmtc.WarningLn("Set up the project")
		fmtc.ToDoLn("Run madock setup")
	}
}

func getArgs() *ArgsStruct {
	args := new(ArgsStruct)
	if len(os.Args) > 2 {
		argsOrigin := os.Args[2:]
		var err error
		_, err = flags.ParseArgs(args, argsOrigin)

		if err != nil {
			log.Fatal(err)
		}
	}

	return args
}
