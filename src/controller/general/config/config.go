package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/faradey/madock/v3/src/command"
	"github.com/faradey/madock/v3/src/helper/cli/arg_struct"
	"github.com/faradey/madock/v3/src/helper/cli/attr"
	"github.com/faradey/madock/v3/src/helper/cli/output"
	"github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/logger"
	"github.com/faradey/madock/v3/src/helper/paths"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"config:cache:clean", "c:c:c"},
		Handler:  CacheClean,
		Help:     "Clean config cache",
		Category: "config",
	})
	command.Register(&command.Definition{
		Aliases:  []string{"config:list"},
		Handler:  ShowEnv,
		Help:     "List configuration. Supports --json (-j) output",
		Category: "config",
	})
	command.Register(&command.Definition{
		Aliases:  []string{"config:set"},
		Handler:  SetEnvOption,
		Help:     "Set configuration option",
		Category: "config",
	})
}

type ConfigListOutput struct {
	Project string            `json:"project"`
	Config  map[string]string `json:"config"`
}

func ShowEnv() {
	args := attr.Parse(new(arg_struct.ControllerGeneralConfigList)).(*arg_struct.ControllerGeneralConfigList)

	projectName := configs.GetProjectName()
	lines := configs.GetProjectConfig(projectName)

	if args.Json {
		output.PrintJSON(ConfigListOutput{
			Project: projectName,
			Config:  lines,
		})
		return
	}

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
