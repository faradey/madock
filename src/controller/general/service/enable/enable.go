package enable

import (
	"fmt"
	"github.com/faradey/madock/src/controller/general/rebuild"
	"github.com/faradey/madock/src/controller/general/service"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/configs"
	"os"
	"os/exec"
)

type ArgsStruct struct {
	attr.ArgumentsWithArgs
	Global bool `arg:"-g,--global" help:"Global"`
}

func Execute() {
	args := attr.Parse(new(ArgsStruct)).(*ArgsStruct)

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
