package prune

import (
	"github.com/alexflint/go-arg"
	"github.com/faradey/madock/src/controller/general/proxy"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/docker"
	"github.com/faradey/madock/src/helper/paths"
	"log"
	"os"
)

type ArgsStruct struct {
	attr.Arguments
	WithVolumes bool `arg:"-v,--with-volumes" help:"With Volumes"`
}

func Execute() {
	args := getArgs()

	if !configs.IsHasNotConfig() {
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

func getArgs() *ArgsStruct {
	args := new(ArgsStruct)
	if attr.IsParseArgs && len(os.Args) > 2 {
		argsOrigin := os.Args[2:]
		p, err := arg.NewParser(arg.Config{
			IgnoreEnv: true,
		}, args)

		if err != nil {
			log.Fatal(err)
		}

		err = p.Parse(argsOrigin)

		if err != nil {
			log.Fatal(err)
		}
	}

	attr.IsParseArgs = false
	return args
}
