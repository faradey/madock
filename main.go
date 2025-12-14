package main

import (
	"log"
	"os"
	"strings"

	"github.com/faradey/madock/src/command"
	_ "github.com/faradey/madock/src/command" // Register commands via init()
	"github.com/faradey/madock/src/controller/general/help"
	"github.com/faradey/madock/src/controller/general/isnotdefine"
	"github.com/faradey/madock/src/migration"
)

var appVersion = "3.1.0"

func main() {
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

//TODO check rabbitMQ in browser
//TODO add new argument "--name, -n" to "madock project:remove" CLI command
