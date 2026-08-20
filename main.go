package main

import (
	dockerassets "github.com/faradey/madock/v4/docker"
	scriptassets "github.com/faradey/madock/v4/scripts"
	"github.com/faradey/madock/v4/src/app"
	"github.com/faradey/madock/v4/src/helper/embedded"
	"github.com/faradey/madock/v4/src/version"

	// Register all built-in controllers
	_ "github.com/faradey/madock/v4/src/controller/all"
)

func main() {
	embedded.SetDockerFS(dockerassets.FS)
	embedded.SetScriptsFS(scriptassets.FS)
	app.Run(version.Version)
}
