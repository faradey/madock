package enable

import (
	"fmt"
	"github.com/alexflint/go-arg"
	"github.com/faradey/madock/src/controller/general/rebuild"
	"github.com/faradey/madock/src/controller/general/service"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/configs"
	"log"
	"os"
	"os/exec"
)

type ArgsStruct struct {
	attr.ArgumentsWithArgs
	Global bool `arg:"-g,--global" help:"Global"`
}

func Execute() {
	args := getArgs()

	if len(args.Args) > 0 {
		for _, name := range args.Args {
			if service.IsService(name) {
				serviceName := service.GetByShort(name) + "/enabled"
				projectName := configs.GetProjectName()
				projectConfig := configs.GetProjectConfig(projectName)
				configs.SetParam(projectName, serviceName, "true", projectConfig["activeScope"])

				if args.Global {
					configs.SetParam(configs.MainConfigCode, serviceName, "true", "default")
				}

			}
		}
	}

	rebuild.Execute()
	cmd := exec.Command("cd", "../")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmt.Println(err)
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
