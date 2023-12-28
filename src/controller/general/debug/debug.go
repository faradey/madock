package debug

import (
	"github.com/alexflint/go-arg"
	"github.com/faradey/madock/src/controller/general/rebuild"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/paths"
	"log"
	"os"
)

type ArgsStruct struct {
	attr.Arguments
}

func Enable() {
	getArgs()
	configPath := paths.GetExecDirPath() + "/projects/" + configs.GetProjectName() + "/config.xml"
	configs.SetParam(configPath, "php/xdebug/enabled", "true", configs.GetCurrentProjectConfig()["activeScope"])
	rebuild.Execute()
}

func Disable() {
	getArgs()
	configs.SetParam(configs.GetProjectName(), "php/xdebug/enabled", "false", configs.GetCurrentProjectConfig()["activeScope"])
	rebuild.Execute()
}

func ProfileEnable() {
	getArgs()
	configs.SetParam(configs.GetProjectName(), "php/xdebug/mode", "profile", configs.GetCurrentProjectConfig()["activeScope"])
	rebuild.Execute()
}

func ProfileDisable() {
	getArgs()
	configs.SetParam(configs.GetProjectName(), "php/xdebug/mode", "debug", configs.GetCurrentProjectConfig()["activeScope"])
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
