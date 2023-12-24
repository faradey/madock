package debug

import (
	"github.com/alexflint/go-arg"
	"github.com/faradey/madock/src/controller/general/rebuild"
	"github.com/faradey/madock/src/helper/cli/attr"
	configs2 "github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/paths"
	"log"
	"os"
)

type ArgsStruct struct {
	attr.Arguments
}

func Enable() {
	getArgs()
	configPath := paths.GetExecDirPath() + "/projects/" + configs2.GetProjectName() + "/env.txt"
	configs2.SetParam(configPath, "XDEBUG_ENABLED", "true")
	rebuild.Execute()
}

func Disable() {
	getArgs()
	configPath := paths.GetExecDirPath() + "/projects/" + configs2.GetProjectName() + "/env.txt"
	configs2.SetParam(configPath, "XDEBUG_ENABLED", "false")
	rebuild.Execute()
}

func ProfileEnable() {
	getArgs()
	configPath := paths.GetExecDirPath() + "/projects/" + configs2.GetProjectName() + "/env.txt"
	configs2.SetParam(configPath, "XDEBUG_MODE", "profile")
	rebuild.Execute()
}

func ProfileDisable() {
	getArgs()
	configPath := paths.GetExecDirPath() + "/projects/" + configs2.GetProjectName() + "/env.txt"
	configs2.SetParam(configPath, "XDEBUG_MODE", "debug")
	rebuild.Execute()
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