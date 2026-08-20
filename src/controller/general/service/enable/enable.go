package enable

import (
	"github.com/faradey/madock/v4/src/command"
	"github.com/faradey/madock/v4/src/controller/general/rebuild"
	"github.com/faradey/madock/v4/src/controller/general/service"
	"github.com/faradey/madock/v4/src/helper/cli/arg_struct"
	"github.com/faradey/madock/v4/src/helper/cli/attr"
	"github.com/faradey/madock/v4/src/helper/cli/fmtc"
	"github.com/faradey/madock/v4/src/helper/configs"
	"github.com/faradey/madock/v4/src/helper/setup/tools"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"service:enable"},
		Handler:  Execute,
		Help:     "Enable service",
		Category: "service",
		ArgsType: new(arg_struct.ControllerGeneralServiceEnable),
	})
}

// versionPrompts maps a service short name to the tools function that
// interactively asks the user for a version. Only services that support
// version selection at enable time appear here.
var versionPrompts = map[string]func(*string){
	"valkey":    tools.Valkey,
	"memcached": tools.Memcached,
	"artemis":   tools.Artemis,
	"xdebug":    tools.Xdebug,
}

func Execute() {
	args := attr.Parse(new(arg_struct.ControllerGeneralServiceEnable)).(*arg_struct.ControllerGeneralServiceEnable)

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

		if !service.IsService(name) {
			continue
		}
		configKey := service.GetByShort(name)
		serviceName := service.ConfigKeyOf(name)
		projectName := configs.GetProjectName()
		projectConfig := configs.GetProjectConfig(projectName)
		configs.SetParam(projectName, serviceName, "true", projectConfig["activeScope"], "")

		if args.Global {
			configs.SetParam(projectName, serviceName, "true", "default", configs.MainConfigCode)
		}

		// If the service supports version selection, either accept --version
		// or prompt interactively, then persist <service>/version.
		if prompt, ok := versionPrompts[name]; ok {
			versionKey := configKey + "/version"
			version := args.Version
			if version == "" {
				version = projectConfig[versionKey]
				prompt(&version)
			}
			if version != "" {
				configs.SetParam(projectName, versionKey, version, projectConfig["activeScope"], "")
				if args.Global {
					configs.SetParam(projectName, versionKey, version, "default", configs.MainConfigCode)
				}
			}
		}
	}

	rebuild.Execute()
}
