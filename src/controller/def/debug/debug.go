package debug

import (
	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/controller/def/rebuild"
	"github.com/faradey/madock/src/paths"
)

func Enable() {
	configPath := paths.GetExecDirPath() + "/projects/" + configs.GetProjectName() + "/env.txt"
	configs.SetParam(configPath, "XDEBUG_ENABLED", "true")
	rebuild.Execute()
}

func Disable() {
	configPath := paths.GetExecDirPath() + "/projects/" + configs.GetProjectName() + "/env.txt"
	configs.SetParam(configPath, "XDEBUG_ENABLED", "false")
	rebuild.Execute()
}

func ProfileEnable() {
	configPath := paths.GetExecDirPath() + "/projects/" + configs.GetProjectName() + "/env.txt"
	configs.SetParam(configPath, "XDEBUG_MODE", "profile")
	rebuild.Execute()
}

func ProfileDisable() {
	configPath := paths.GetExecDirPath() + "/projects/" + configs.GetProjectName() + "/env.txt"
	configs.SetParam(configPath, "XDEBUG_MODE", "debug")
	rebuild.Execute()
}
