package debug

import (
	"github.com/faradey/madock/v4/src/command"
	"github.com/faradey/madock/v4/src/controller/general/rebuild"
	"github.com/faradey/madock/v4/src/helper/cli/attr"
	"github.com/faradey/madock/v4/src/helper/cli/fmtc"
	"github.com/faradey/madock/v4/src/helper/configs"
)

type ArgsStruct struct {
	attr.Arguments
}

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"debug:enable"},
		Handler:  Enable,
		Help:     "Enable debug mode",
		Category: "debug",
		ArgsType: new(ArgsStruct),
	})
	command.Register(&command.Definition{
		Aliases:  []string{"debug:disable"},
		Handler:  Disable,
		Help:     "Disable debug mode",
		Category: "debug",
		ArgsType: new(ArgsStruct),
	})
	command.Register(&command.Definition{
		Aliases:  []string{"debug:profile:enable"},
		Handler:  ProfileEnable,
		Help:     "Enable profiler",
		Category: "debug",
		ArgsType: new(ArgsStruct),
	})
	command.Register(&command.Definition{
		Aliases:  []string{"debug:profile:disable"},
		Handler:  ProfileDisable,
		Help:     "Disable profiler",
		Category: "debug",
		ArgsType: new(ArgsStruct),
	})
}

// requirePHP stops the profiler commands on a project they cannot do anything
// for.
//
// They write php/xdebug/mode, so on a project with no PHP container they set a
// value nothing reads, rebuild, and report success. Profiling is then simply
// absent, and the command that was supposed to arrange it said it had. The
// profiler stayed PHP-only when debugging stopped being: `--cpu-prof` writes a
// file for somebody to open afterwards, which is a different command with a
// different answer, not this one with a second branch.
func requirePHP() bool {
	projectConf := configs.GetCurrentProjectConfig()
	if projectConf["php/enabled"] == "true" {
		return true
	}

	language := projectConf["language"]
	if language == "" {
		language = "this project's language"
	}
	fmtc.ErrorLn("Profiling is only wired up for PHP, and " + language + " has no PHP container.")
	fmtc.ToDoLn("Nothing was changed.")
	return false
}

// runtime is something in this project that can be put into debug mode, and the
// switch that does it.
type runtime struct {
	name string
	key  string
}

// debuggable lists the runtimes this project actually has.
//
// Deliberately not a single `debug/enabled` key derived from `language`: a
// project can run more than one of these at once — a PHP application with a
// JavaScript front end in its own container is the ordinary case — and one
// switch would have had to pick a winner. The per-runtime keys already exist
// for other purposes, so the command derives the list and leaves the
// configuration alone; php/xdebug/* keeps the name it has always had, and
// nothing that reads it notices this change.
//
// Kept free of I/O so the choice can be tested without a project on disk.
func debuggable(projectConf map[string]string) []runtime {
	var found []runtime

	if projectConf["php/enabled"] == "true" {
		found = append(found, runtime{name: "PHP (xdebug)", key: "php/xdebug/enabled"})
	}
	if projectConf["nodejs/enabled"] == "true" {
		found = append(found, runtime{name: "Node", key: "nodejs/debug/enabled"})
	}

	return found
}

// setDebug writes the switch for every debuggable runtime and rebuilds once.
//
// Nothing is silent: a project with two runtimes says so, because "debugging is
// on" without saying what for is how somebody ends up attaching to the wrong
// container.
func setDebug(value string) {
	projectConf := configs.GetCurrentProjectConfig()

	runtimes := debuggable(projectConf)
	if len(runtimes) == 0 {
		refuse(projectConf)
		return
	}

	for _, r := range runtimes {
		fmtc.SuccessLn(verb(value) + " debugging for " + r.name)
		configs.SetParam(configs.GetProjectName(), r.key, value, projectConf["activeScope"], "")
	}

	// Node's debugger listens, so the port only exists once the stack has been
	// generated again — which is what makes the number worth pointing at rather
	// than printing here, where it would be the previous rebuild's.
	if value == "true" && projectConf["nodejs/enabled"] == "true" {
		fmtc.ToDoLn("madock info:ports — the debugger listens on the port published for nodejs_debug")
	}

	rebuild.Execute()
}

func verb(value string) string {
	if value == "true" {
		return "Enabling"
	}
	return "Disabling"
}

// refuse says why nothing happened, and names the one case that looks like it
// should have worked.
func refuse(projectConf map[string]string) {
	language := projectConf["language"]
	if language == "" {
		language = "this project's language"
	}

	fmtc.ErrorLn("Debugging is wired up for PHP and Node, and this project runs neither: " + language + ".")

	// Node in the application container has no container of its own and so no
	// port to publish, which is the whole mechanism its debugger needs. Somebody
	// who has switched it on has every reason to expect this command to work.
	if projectConf["nodejs/embedded/enabled"] == "true" {
		fmtc.WarningLn("Node here runs inside the application container (nodejs/embedded/enabled), " +
			"and its debugger needs a published port, which only a container of its own has.")
		fmtc.ToDoLn("madock config:set -n nodejs/enabled -v true to give it one")
	}

	fmtc.ToDoLn("Nothing was changed.")
}

func Enable() {
	attr.Parse(new(ArgsStruct))
	setDebug("true")
}

func Disable() {
	attr.Parse(new(ArgsStruct))
	setDebug("false")
}

func ProfileEnable() {
	attr.Parse(new(ArgsStruct))
	if !requirePHP() {
		return
	}
	configs.SetParam(configs.GetProjectName(), "php/xdebug/mode", "profile", configs.GetCurrentProjectConfig()["activeScope"], "")
	rebuild.Execute()
}

func ProfileDisable() {
	attr.Parse(new(ArgsStruct))
	if !requirePHP() {
		return
	}
	configs.SetParam(configs.GetProjectName(), "php/xdebug/mode", "debug", configs.GetCurrentProjectConfig()["activeScope"], "")
	rebuild.Execute()
}
