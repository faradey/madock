package config

import (
	"fmt"
	"github.com/alexflint/go-arg"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/paths"
	"log"
	"os"
	"strings"
)

type ArgsStruct struct {
	attr.Arguments
	Name  string `arg:"-n,--name" help:"Parameter name"`
	Value string `arg:"-v,--value" help:"Parameter value"`
	Scope string `arg:"-s,--scope" help:"Scope name"`
}

func ShowEnv() {
	lines := configs.GetProjectConfig(configs.GetProjectName())
	for key, line := range lines {
		fmt.Println(key + " " + line)
	}
}

func SetEnvOption() {
	args := getArgs()
	name := strings.ToUpper(args.Name)
	val := args.Value
	activeScope := args.Scope
	if len(activeScope) == 0 {
		activeScope = "default"
	}
	if len(name) > 0 && configs.IsOption(name) {
		configPath := paths.GetExecDirPath() + "/projects/" + configs.GetProjectName() + "/config.xml"
		configs.SetParam(configPath, name, val, activeScope)
	}
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
