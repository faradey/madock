package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/faradey/madock/v4/src/command"
	"github.com/faradey/madock/v4/src/helper/cli/arg_struct"
	"github.com/faradey/madock/v4/src/helper/cli/attr"
	"github.com/faradey/madock/v4/src/helper/cli/fmtc"
	"github.com/faradey/madock/v4/src/helper/cli/output"
	"github.com/faradey/madock/v4/src/helper/configs"
	"github.com/faradey/madock/v4/src/helper/logger"
	"github.com/faradey/madock/v4/src/helper/paths"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"config:cache:clean", "c:c:c"},
		Handler:  CacheClean,
		Help:     "Clean config cache",
		Category: "config",
	})
	command.Register(&command.Definition{
		Aliases:    []string{"config:list"},
		JSONOutput: true,
		Handler:    ShowEnv,
		Help:       "List configuration. Supports --json (-j) output",
		Category:   "config",
		ArgsType:   new(arg_struct.ControllerGeneralConfigList),
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
		projectName := configs.GetProjectName()
		configs.SetParam(projectName, name, val, activeScope, "")
		dropStaleDerived(projectName, name, activeScope)
	}
}

// dropStaleDerived takes the stored copy of a computed key out of the file when
// the key it is computed from is assigned.
//
// A derived value is recomputed on every read, so a copy in a file can only go
// stale — and it is read by people even when nothing reads it. Measured on a
// live server: `config:set nodejs/version 22.22.0` left `major_version 20` in
// the file, and the next reader concluded the environment would build Node 20.
// It would not, because the render derives 22 from the version — but the file
// said otherwise and the file is what people check.
func dropStaleDerived(projectName, source, activeScope string) {
	file := paths.GetExecDirPath() + "/projects/" + projectName + "/config.xml"
	removed, err := configs.RemoveStoredDerived(file, source, activeScope)
	if err != nil {
		fmtc.WarningLn("Could not clear the stored value of " + strings.Join(configs.DerivedFrom(source), ", ") + ": " + err.Error())
		return
	}
	if len(removed) == 0 {
		return
	}
	configs.CleanCache()

	for _, key := range removed {
		fmtc.SuccessLn("Cleared the stored \"" + key + "\" — it is computed from \"" + source + "\" on every read.")
	}

	// Read the one other file that stores it and that this command does not
	// write. Asking the merged config would answer nothing: it recomputes the
	// derived key on every read, so the key is always there and always right.
	// What matters is whether a *stored* copy survives, and only the file says.
	projectFile := paths.GetRunDirPath() + "/.madock/config.xml"
	if !paths.IsFileExist(projectFile) {
		return
	}
	stored := configs.ParseXmlFile(projectFile)
	for _, key := range removed {
		if value, still := stored["scopes/"+activeScope+"/"+key]; still {
			fmtc.WarningLn("\"" + key + "\" is still written as \"" + value + "\" in the project's own .madock/config.xml.")
			fmtc.ToDoLn("Remove it there too — it is derived, so a stored copy can only go stale. That file is committed, so this command does not touch it.")
		}
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
