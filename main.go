package main

import (
	"github.com/faradey/madock/v3/src/app"

	// Register all built-in controllers
	_ "github.com/faradey/madock/v3/src/controller/all"
)

var appVersion = "3.3.0"

func main() {
	app.Run(appVersion)
}
