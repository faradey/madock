package debug

import (
	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/controller/general/rebuild"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/paths"
	"github.com/jessevdk/go-flags"
	"log"
	"os"
)

type ArgsStruct struct {
	attr.Arguments
}

func Enable() {
	getArgs()
	configPath := paths.GetExecDirPath() + "/projects/" + configs.GetProjectName() + "/env.txt"
	configs.SetParam(configPath, "XDEBUG_ENABLED", "true")
	rebuild.Execute()
}

func Disable() {
	getArgs()
	configPath := paths.GetExecDirPath() + "/projects/" + configs.GetProjectName() + "/env.txt"
	configs.SetParam(configPath, "XDEBUG_ENABLED", "false")
	rebuild.Execute()
}

func ProfileEnable() {
	getArgs()
	configPath := paths.GetExecDirPath() + "/projects/" + configs.GetProjectName() + "/env.txt"
	configs.SetParam(configPath, "XDEBUG_MODE", "profile")
	rebuild.Execute()
}

func ProfileDisable() {
	getArgs()
	configPath := paths.GetExecDirPath() + "/projects/" + configs.GetProjectName() + "/env.txt"
	configs.SetParam(configPath, "XDEBUG_MODE", "debug")
	rebuild.Execute()
}

func getArgs() *ArgsStruct {
	args := new(ArgsStruct)
	if len(os.Args) > 2 {
		argsOrigin := os.Args[2:]
		var err error
		_, err = flags.ParseArgs(args, argsOrigin)

		if err != nil {
			log.Fatal(err)
		}
	}

	return args
}
