package config

import (
	"fmt"
	"github.com/faradey/madock/src/cli/attr"
	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/paths"
	"github.com/jessevdk/go-flags"
	"log"
	"os"
	"strings"
)

type ArgsStruct struct {
	attr.Arguments
	Name  string `long:"name" description:"Parameter name"`
	Value string `long:"value" description:"Parameter value"`
}

func ShowEnv() {
	configPath := paths.GetExecDirPath() + "/projects/" + configs.GetProjectName() + "/env.txt"
	lines := configs.GetAllLines(configPath)
	for _, ln := range lines {
		fmt.Println(ln)
	}
}

func SetEnvOption() {
	args := getArgs()
	name := strings.ToUpper(args.Name)
	val := args.Value
	if len(name) > 0 && configs.IsOption(name) {
		configPath := paths.GetExecDirPath() + "/projects/" + configs.GetProjectName() + "/env.txt"
		configs.SetParam(configPath, name, val)
	}
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
