package enable

import (
	"github.com/faradey/madock/v3/src/command"
	"github.com/faradey/madock/v3/src/controller/general/rebuild"
	"github.com/faradey/madock/v3/src/controller/general/service"
	"github.com/faradey/madock/v3/src/helper/cli/arg_struct"
	"github.com/faradey/madock/v3/src/helper/cli/attr"
	"github.com/faradey/madock/v3/src/helper/cli/fmtc"
	"github.com/faradey/madock/v3/src/helper/configs"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"service:enable"},
		Handler:  Execute,
		Help:     "Enable service",
		Category: "service",
	})
}

func Execute() {
	args := attr.Parse(new(arg_struct.ControllerGeneralServiceEnable)).(*arg_struct.ControllerGeneralServiceEnable)

	if len(args.Args) == 0 {
		fmtc.ErrorLn("Service name(s) is required")
		return
	}

	for _, name := range args.Args {
		if service.IsService(name) {
			serviceName := service.GetByShort(name) + "/enabled"
			projectName := configs.GetProjectName()
			projectConfig := configs.GetProjectConfig(projectName)
			configs.SetParam(projectName, serviceName, "true", projectConfig["activeScope"], "")

			if args.Global {
				configs.SetParam(projectName, serviceName, "true", "default", configs.MainConfigCode)
			}
		}
	}

	rebuild.Execute()
}
