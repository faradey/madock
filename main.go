package main

import (
	"github.com/faradey/madock/src/app"

	// Register all built-in controllers
	_ "github.com/faradey/madock/src/controller/all"
)

var appVersion = "3.3.0"

func main() {
	app.Run(appVersion)
}
