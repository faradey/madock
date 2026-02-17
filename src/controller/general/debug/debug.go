package debug

import (
	"github.com/faradey/madock/v3/src/command"
	"github.com/faradey/madock/v3/src/controller/general/rebuild"
	"github.com/faradey/madock/v3/src/helper/cli/attr"
	"github.com/faradey/madock/v3/src/helper/configs"
)

type ArgsStruct struct {
	attr.Arguments
}

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"debug:enable"},
		Handler:  Enable,
		Help:     "Enable debug mode",
		Category: "debug",
	})
	command.Register(&command.Definition{
		Aliases:  []string{"debug:disable"},
		Handler:  Disable,
		Help:     "Disable debug mode",
		Category: "debug",
	})
	command.Register(&command.Definition{
		Aliases:  []string{"debug:profile:enable"},
		Handler:  ProfileEnable,
		Help:     "Enable profiler",
		Category: "debug",
	})
	command.Register(&command.Definition{
		Aliases:  []string{"debug:profile:disable"},
		Handler:  ProfileDisable,
		Help:     "Disable profiler",
		Category: "debug",
	})
}

func Enable() {
	attr.Parse(new(ArgsStruct))
	configs.SetParam(configs.GetProjectName(), "php/xdebug/enabled", "true", configs.GetCurrentProjectConfig()["activeScope"], "")
	rebuild.Execute()
}

func Disable() {
	attr.Parse(new(ArgsStruct))
	configs.SetParam(configs.GetProjectName(), "php/xdebug/enabled", "false", configs.GetCurrentProjectConfig()["activeScope"], "")
	rebuild.Execute()
}

func ProfileEnable() {
	attr.Parse(new(ArgsStruct))
	configs.SetParam(configs.GetProjectName(), "php/xdebug/mode", "profile", configs.GetCurrentProjectConfig()["activeScope"], "")
	rebuild.Execute()
}

func ProfileDisable() {
	attr.Parse(new(ArgsStruct))
	configs.SetParam(configs.GetProjectName(), "php/xdebug/mode", "debug", configs.GetCurrentProjectConfig()["activeScope"], "")
	rebuild.Execute()
}
