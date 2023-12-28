package config

import (
	"fmt"
	"github.com/alexflint/go-arg"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/configs"
	"log"
	"os"
	"strings"
)

type ArgsStruct struct {
	attr.Arguments
	Name  string `arg:"-n,--name" help:"Parameter name"`
	Value string `arg:"-v,--value" help:"Parameter value"`
}

func ShowEnv() {
	lines := configs.GetProjectConfig(configs.GetProjectName())
	for key, line := range lines {
		fmt.Println(key + " " + line)
	}
}

func SetEnvOption() {
	args := getArgs()
	name := strings.ToLower(args.Name)
	val := args.Value
	activeScope := "default"
	projectConfig := configs.GetCurrentProjectConfig()
	if _, ok := projectConfig["activeScope"]; ok {
		activeScope = projectConfig["activeScope"]
	}
	if len(name) > 0 && configs.IsOption(name) {
		configs.SetParam(configs.GetProjectName(), name, val, activeScope)
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
