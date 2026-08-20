package stop

import (
	"github.com/faradey/madock/v4/src/command"
	"github.com/faradey/madock/v4/src/controller/general/proxy"
	"github.com/faradey/madock/v4/src/controller/platform"
	"github.com/faradey/madock/v4/src/helper/configs"
	"github.com/faradey/madock/v4/src/helper/paths"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"stop"},
		Handler:  Execute,
		Help:     "Stop containers",
		Category: "general",
	})
}

func Execute() {
	projectName := configs.GetProjectName()
	projectConf := configs.GetCurrentProjectConfig()
	platformName := projectConf["platform"]

	handler := platform.GetOrDefault(platformName)
	handler.Stop(projectName)

	if len(paths.GetActiveProjects()) == 0 {
		proxy.Execute("stop")
	}
}
