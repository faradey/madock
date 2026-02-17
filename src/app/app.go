package app

import (
	"log"
	"os"
	"strings"

	"github.com/faradey/madock/src/command"
	"github.com/faradey/madock/src/controller/general/help"
	"github.com/faradey/madock/src/controller/general/isnotdefine"
	"github.com/faradey/madock/src/migration"
)

// Run is the main entry point for the madock application.
// It applies migrations, parses the command from os.Args, and dispatches
// to the appropriate registered handler.
func Run(appVersion string) {
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.Lmicroseconds)
	migration.Apply(appVersion)

	if len(os.Args) <= 1 {
		help.Execute()
		return
	}

	cmdName := strings.ToLower(os.Args[1])

	if def, ok := command.Get(cmdName); ok {
		def.Handler()
	} else {
		isnotdefine.Execute(cmdName)
	}
}
