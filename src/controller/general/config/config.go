package config

import (
	"fmt"
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
}

func ShowEnv() {
	lines := configs.GetProjectConfig(configs.GetProjectName())
	for key, line := range lines {
		fmt.Println(key + " " + line)
	}
}

func SetEnvOption() {
	args := attr.Parse(new(ArgsStruct)).(*ArgsStruct)
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

func CacheClean() {
	folder := paths.MakeDirsByPath(paths.GetExecDirPath() + "/cache/")
	err := os.RemoveAll(folder)
	if err != nil {
		log.Fatal(err)
	}
	paths.MakeDirsByPath(paths.GetExecDirPath() + "/cache/")
}
