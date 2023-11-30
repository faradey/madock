package rebuild

import (
	"github.com/faradey/madock/src/controller/general/proxy"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/docker"
	"github.com/faradey/madock/src/helper/paths"
	"github.com/jessevdk/go-flags"
	"log"
	"os"
)

type ArgsStruct struct {
	attr.Arguments
	Force     bool `long:"force" short:"f" description:"Force"`
	WithChown bool `long:"with-chown" short:"c" description:"With Chown"`
}

func Execute() {
	args := getArgs()

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
