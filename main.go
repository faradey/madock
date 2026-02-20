package main

import (
	"github.com/faradey/madock/v3/src/app"
	"github.com/faradey/madock/v3/src/version"

	// Register all built-in controllers
	_ "github.com/faradey/madock/v3/src/controller/all"
)

func main() {
	app.Run(version.Version)
}
