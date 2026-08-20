package app

import (
	"log"
	"os"
	"strings"

	"github.com/faradey/madock/v3/src/command"
	"github.com/faradey/madock/v3/src/controller/general/help"
	"github.com/faradey/madock/v3/src/controller/general/isnotdefine"
	"github.com/faradey/madock/v3/src/helper/cli/fmtc"
	"github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/embedded"
	"github.com/faradey/madock/v3/src/helper/paths"
	"github.com/faradey/madock/v3/src/migration"
)

// Run is the main entry point for the madock application.
// It applies migrations, parses the command from os.Args, and dispatches
// to the appropriate registered handler.
func Run(appVersion string) {
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.Lmicroseconds)
	embedded.ExtractIfNeeded(appVersion)
	migration.Apply(appVersion)

	if len(os.Args) <= 1 {
		help.Execute()
		return
	}

	cmdName := strings.ToLower(os.Args[1])

	if def, ok := command.Get(cmdName); ok {
		// Before the project check, and before the handler. Asking what a command
		// does is not itself a use of a project, so it answers outside one too.
		if !def.PassThrough && wantsHelp(os.Args[2:]) {
			help.ShowCommand(cmdName)
			return
		}

		if !command.IsGlobal(def) && !configs.IsHasConfig("") {
			refuseOutsideProject(cmdName)
			return
		}
		def.Handler()
	} else {
		isnotdefine.Execute(cmdName)
	}
}

// wantsHelp reports whether the arguments ask for the command's help.
//
// `--help` only, never `-h`. The short form is a real flag with a real value
// elsewhere — `setup:env -h <host>` — and a dispatcher cannot tell the two apart
// without knowing the command's own flags. Where a command has an argument
// struct, go-arg still answers -h from inside it; where it has none there is
// nothing for -h to collide with, and nothing for it to mean either.
//
// A `--` ends it, because everything after that belongs to whatever the command
// passes it to.
func wantsHelp(argv []string) bool {
	for _, arg := range argv {
		if arg == "--" {
			return false
		}
		if arg == "--help" {
			return true
		}
	}

	return false
}

// refuseOutsideProject explains why a project command will not run here.
//
// The project name comes from the name of the current directory, so without this
// a command in a directory that is not a project quietly adopted any generated
// files sitting under that name. In the madock source tree itself — a directory
// called "madock", next to runtime leftovers from an older version — `restart`
// therefore ran docker compose against a file from months earlier and failed on an
// empty host name, blaming a service in a project that does not exist.
func refuseOutsideProject(cmdName string) {
	name := configs.GetProjectName()

	fmtc.ErrorLn("This directory is not a madock project, so " + cmdName + " has nothing to act on")

	// A runtime directory without a configuration is the leftover case, and worth
	// naming: it is what made the old behaviour look like a working project.
	//
	// Deliberately not pointing at project:remove. That command derives the
	// project from the current directory and finishes with RemoveAll on it, so on
	// a leftover whose configuration is gone it would delete the directory the
	// person is standing in — and one such leftover is the madock installation
	// itself, whose runtime `src` is a symlink to the source tree.
	leftover := paths.GetExecDirPath() + "/aruntime/projects/" + name
	if paths.IsFileExist(leftover) {
		fmtc.WarningLn("There are leftover generated files for the name '" + name + "' from an earlier project:")
		fmtc.WarningLn("  " + leftover)
		fmtc.ToDoLn("Delete that directory, or run madock setup to make this directory a project")
		os.Exit(1)
	}

	fmtc.ToDoLn("Run madock setup")
	os.Exit(1)
}
