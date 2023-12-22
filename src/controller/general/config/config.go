package config

import (
	"fmt"
	"github.com/faradey/madock/src/helper/cli/attr"
	configs2 "github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/paths"
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
	configPath := paths.GetExecDirPath() + "/projects/" + configs2.GetProjectName() + "/env.txt"
	lines := configs2.GetAllLines(configPath)
	for _, ln := range lines {
		fmt.Println(ln)
	}
}

func SetEnvOption() {
	args := getArgs()
	name := strings.ToUpper(args.Name)
	val := args.Value
	if len(name) > 0 && configs2.IsOption(name) {
		configPath := paths.GetExecDirPath() + "/projects/" + configs2.GetProjectName() + "/env.txt"
		configs2.SetParam(configPath, name, val)
	}
}

func getArgs() *ArgsStruct {
	args := new(ArgsStruct)
	if attr.IsParseArgs && len(os.Args) > 2 {
		argsOrigin := os.Args[2:]
		var err error
		_, err = flags.ParseArgs(args, argsOrigin)

		if err != nil {
			log.Fatal(err)
		}
	}

	attr.IsParseArgs = false
	return args
}
