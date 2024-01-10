package debug

import (
	"github.com/faradey/madock/src/controller/general/rebuild"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/configs"
)

type ArgsStruct struct {
	attr.Arguments
}

func Enable() {
	attr.Parse(new(ArgsStruct))
	configs.SetParam(configs.GetProjectName(), "php/xdebug/enabled", "true", configs.GetCurrentProjectConfig()["activeScope"])
	rebuild.Execute()
}

func Disable() {
	attr.Parse(new(ArgsStruct))
	configs.SetParam(configs.GetProjectName(), "php/xdebug/enabled", "false", configs.GetCurrentProjectConfig()["activeScope"])
	rebuild.Execute()
}

func ProfileEnable() {
	attr.Parse(new(ArgsStruct))
	configs.SetParam(configs.GetProjectName(), "php/xdebug/mode", "profile", configs.GetCurrentProjectConfig()["activeScope"])
	rebuild.Execute()
}

func ProfileDisable() {
	attr.Parse(new(ArgsStruct))
	configs.SetParam(configs.GetProjectName(), "php/xdebug/mode", "debug", configs.GetCurrentProjectConfig()["activeScope"])
	rebuild.Execute()
}
