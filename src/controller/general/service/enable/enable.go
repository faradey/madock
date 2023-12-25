package enable

import (
	"github.com/alexflint/go-arg"
	"github.com/faradey/madock/src/controller/general/rebuild"
	"github.com/faradey/madock/src/controller/general/service"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/paths"
	"log"
	"os"
	"strings"
)

type ArgsStruct struct {
	attr.ArgumentsWithArgs
	Global bool `arg:"-g,--global" help:"Global"`
}

func Execute() {
	args := getArgs()

	if len(args.Args) > 0 {
		for _, name := range args.Args {
			name = strings.ToLower(name)
			if service.IsService(name) {
				serviceName := strings.ToLower(name) + "/enabled"
				projectName := configs.GetProjectName()
				envFile := paths.MakeDirsByPath(paths.GetExecDirPath()+"/projects/"+projectName) + "/config.xml"
				configs.SetParam(envFile, serviceName, "true")

				if args.Global {
					envFile = paths.MakeDirsByPath(paths.GetExecDirPath()+"/projects") + "/config.xml"
					configs.SetParam(envFile, serviceName, "true")
				}

			}
		}
	}

	rebuild.Execute()
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
