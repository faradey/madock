package disable

import (
	"github.com/faradey/madock/v4/src/command"
	"github.com/faradey/madock/v4/src/controller/general/rebuild"
	"github.com/faradey/madock/v4/src/controller/general/service"
	"github.com/faradey/madock/v4/src/helper/cli/arg_struct"
	"github.com/faradey/madock/v4/src/helper/cli/attr"
	"github.com/faradey/madock/v4/src/helper/cli/fmtc"
	"github.com/faradey/madock/v4/src/helper/configs"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"service:disable"},
		Handler:  Execute,
		Help:     "Disable service",
		Category: "service",
		ArgsType: new(arg_struct.ControllerGeneralServiceDisable),
	})
}

func Execute() {
	args := attr.Parse(new(arg_struct.ControllerGeneralServiceDisable)).(*arg_struct.ControllerGeneralServiceDisable)

	if len(args.Args) == 0 {
		fmtc.ErrorLn("Service name(s) is required")
		return
	}
	for _, name := range args.Args {
		// An old name still works, and says so. Silence here would leave
		// somebody using a name that is one release from disappearing without
		// ever learning there is a new one.
		if current, renamed := service.Renamed(name); renamed {
			fmtc.WarningLn("\"" + name + "\" is now \"" + current + "\". The old name still works for now.")
		}

		if service.IsService(name) {
			serviceName := service.ConfigKeyOf(name)
			projectName := configs.GetProjectName()
			projectConfig := configs.GetProjectConfig(projectName)
			configs.SetParam(projectName, serviceName, "false", projectConfig["activeScope"], "")
			if args.Global {
				configs.SetParam(projectName, serviceName, "false", "default", configs.MainConfigCode)
			}
		}
	}

	rebuild.Execute()
}
