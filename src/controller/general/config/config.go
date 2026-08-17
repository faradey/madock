package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/faradey/madock/v3/src/command"
	"github.com/faradey/madock/v3/src/helper/cli/arg_struct"
	"github.com/faradey/madock/v3/src/helper/cli/attr"
	"github.com/faradey/madock/v3/src/helper/cli/fmtc"
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
		ArgsType: new(arg_struct.ControllerGeneralConfigList),
	})
	command.Register(&command.Definition{
		Aliases:  []string{"config:set"},
		Handler:  SetEnvOption,
		Help:     "Set configuration option",
		Category: "config",
		ArgsType: new(arg_struct.ControllerGeneralConfig),
	})
	command.Register(&command.Definition{
		Aliases:  []string{"config:unset"},
		Handler:  UnsetEnvOption,
		Help:     "Remove a configuration option, so the value underneath it applies again",
		Category: "config",
		ArgsType: new(arg_struct.ControllerGeneralConfigUnset),
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

// UnsetEnvOption removes a setting instead of assigning one.
//
// There was no way to do this at all, and the gap was not obvious because
// nothing failed. madock keeps the project's own `.madock/config.xml` — the
// copy in git — and a machine-side copy under the installation, seeded from it
// the first time the project is seen. Reads merge both, with the project's copy
// winning, so *adding* or *changing* a setting in git reaches every machine.
// **Deleting one does not.** The machine-side copy still has it, `config:set`
// can only assign, and clearing the cache does not touch the file — so the only
// way to drop a single key was to remove the project and set it up again.
//
// Measured on a live project: a `custom_commands` block was deleted from the
// repository, committed and rolled out, and `madock pr` went on working on
// every machine that had ever run setup. Nothing broke and nobody was told,
// which is the whole difficulty — a setting that was retired months ago is
// still in force and nothing anywhere says so.
//
// What this does not decide is which copy should win in general. Making
// `rebuild` reduce the machine-side copy to the project's would fix the class
// rather than the case, and it would also throw away everything set with
// `config:set` on that machine — telling those two apart needs provenance the
// config does not record. This is the smaller, safe half: an explicit removal,
// asked for by name.
func UnsetEnvOption() {
	args := attr.Parse(new(arg_struct.ControllerGeneralConfigUnset)).(*arg_struct.ControllerGeneralConfigUnset)

	if len(args.Name) == 0 {
		fmtc.ErrorLn("Parameter name is required: madock config:unset -n <name>")
		return
	}

	activeScope := "default"
	projectConfig := configs.GetCurrentProjectConfig()
	if scope, ok := projectConfig["activeScope"]; ok && scope != "" {
		activeScope = scope
	}

	projectName := configs.GetProjectName()
	files := []string{paths.GetExecDirPath() + "/projects/" + projectName + "/config.xml"}
	if args.Global {
		files = append(files, paths.GetExecDirPath()+"/config.xml")
	}

	var names []string
	for _, name := range args.Name {
		names = append(names, strings.ToLower(name))
	}

	for _, file := range files {
		if !paths.IsFileExist(file) {
			continue
		}
		if err := configs.RemoveKeepingComments(file, names, activeScope); err != nil {
			logger.Fatalln(err)
		}
	}
	configs.CleanCache()

	// Reported by reading it back, not by trusting the write. A key can survive
	// this in a way that looks exactly like success: it may also be set in the
	// project's own config.xml, which madock does not write, and which wins.
	// Saying so is the difference between a command that worked and a command
	// that appeared to.
	after := configs.GetCurrentProjectConfig()
	for _, name := range names {
		if value, still := after[name]; still {
			fmtc.WarningLn("\"" + name + "\" is still set to \"" + value + "\".")
			fmtc.ToDoLn("It comes from somewhere this command does not write. Check the project's own .madock/config.xml.")
			continue
		}
		fmtc.SuccessLn("Removed \"" + name + "\".")
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
