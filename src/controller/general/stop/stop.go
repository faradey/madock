package stop

import (
	"github.com/faradey/madock/src/controller/general/proxy"
	"github.com/faradey/madock/src/controller/platform"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/paths"
)

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
