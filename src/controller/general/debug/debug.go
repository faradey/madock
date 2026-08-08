package debug

import (
	"github.com/faradey/madock/v3/src/command"
	"github.com/faradey/madock/v3/src/controller/general/rebuild"
	"github.com/faradey/madock/v3/src/helper/cli/attr"
	"github.com/faradey/madock/v3/src/helper/cli/fmtc"
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
		ArgsType: new(ArgsStruct),
	})
	command.Register(&command.Definition{
		Aliases:  []string{"debug:disable"},
		Handler:  Disable,
		Help:     "Disable debug mode",
		Category: "debug",
		ArgsType: new(ArgsStruct),
	})
	command.Register(&command.Definition{
		Aliases:  []string{"debug:profile:enable"},
		Handler:  ProfileEnable,
		Help:     "Enable profiler",
		Category: "debug",
		ArgsType: new(ArgsStruct),
	})
	command.Register(&command.Definition{
		Aliases:  []string{"debug:profile:disable"},
		Handler:  ProfileDisable,
		Help:     "Disable profiler",
		Category: "debug",
		ArgsType: new(ArgsStruct),
	})
}

// requirePHP stops the debug commands on a project they cannot do anything for.
//
// Every one of them writes php/xdebug/*, so on a nodejs, python, golang or ruby
// project they set a value nothing reads, rebuild the project, and report
// success. Debugging is then simply absent, and the command that was supposed
// to arrange it said it had.
//
// The other languages are not wired up yet and that is a piece of work, not an
// oversight — their debuggers listen instead of connecting out, so each needs a
// published port, an allocation, and a process started under the debugger. Until
// then this says so rather than pretending.
func requirePHP() bool {
	projectConf := configs.GetCurrentProjectConfig()
	if projectConf["php/enabled"] == "true" {
		return true
	}

	language := projectConf["language"]
	if language == "" {
		language = "this project's language"
	}
	fmtc.ErrorLn("Debugging is only wired up for PHP, and " + language + " has no PHP container.")
	fmtc.ToDoLn("Nothing was changed.")
	return false
}

func Enable() {
	attr.Parse(new(ArgsStruct))
	if !requirePHP() {
		return
	}
	configs.SetParam(configs.GetProjectName(), "php/xdebug/enabled", "true", configs.GetCurrentProjectConfig()["activeScope"], "")
	rebuild.Execute()
}

func Disable() {
	attr.Parse(new(ArgsStruct))
	if !requirePHP() {
		return
	}
	configs.SetParam(configs.GetProjectName(), "php/xdebug/enabled", "false", configs.GetCurrentProjectConfig()["activeScope"], "")
	rebuild.Execute()
}

func ProfileEnable() {
	attr.Parse(new(ArgsStruct))
	if !requirePHP() {
		return
	}
	configs.SetParam(configs.GetProjectName(), "php/xdebug/mode", "profile", configs.GetCurrentProjectConfig()["activeScope"], "")
	rebuild.Execute()
}

func ProfileDisable() {
	attr.Parse(new(ArgsStruct))
	if !requirePHP() {
		return
	}
	configs.SetParam(configs.GetProjectName(), "php/xdebug/mode", "debug", configs.GetCurrentProjectConfig()["activeScope"], "")
	rebuild.Execute()
}
