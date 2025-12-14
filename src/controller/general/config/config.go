package config

import (
	"fmt"
	"github.com/faradey/madock/src/helper/cli/arg_struct"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/logger"
	"github.com/faradey/madock/src/helper/paths"
	"os"
	"strings"
)

func ShowEnv() {
	lines := configs.GetProjectConfig(configs.GetProjectName())
	for key, line := range lines {
		fmt.Println(key + " " + line)
	}
}

func SetEnvOption() {
	args := attr.Parse(new(arg_struct.ControllerGeneralConfig)).(*arg_struct.ControllerGeneralConfig)
	name := strings.ToLower(args.Name)
	val := args.Value
	activeScope := "default"
	projectConfig := configs.GetCurrentProjectConfig()
	if _, ok := projectConfig["activeScope"]; ok {
		activeScope = projectConfig["activeScope"]
	}
	if len(name) > 0 && configs.IsOption(name) {
		configs.SetParam(configs.GetProjectName(), name, val, activeScope, "")
	}
}

func CacheClean() {
	folder := paths.MakeDirsByPath(paths.CacheDir())
	err := os.RemoveAll(folder)
	if err != nil {
		logger.Fatal(err)
	}
	paths.MakeDirsByPath(paths.CacheDir())
}
