package setup

import (
	"fmt"
	"github.com/faradey/madock/src/controller/shopify/start"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/configs/projects"
	"github.com/faradey/madock/src/helper/docker"
	"github.com/faradey/madock/src/helper/paths"
	"github.com/faradey/madock/src/helper/setup/tools"
	"github.com/faradey/madock/src/model/versions/shopify"
)

func Execute(projectName string, projectConf map[string]string, continueSetup, withVolumes, withChown bool) {
	toolsDefVersions := shopify.GetVersions()

	if continueSetup {
		fmt.Println("")

		tools.Php(&toolsDefVersions.Php)
		tools.Db(&toolsDefVersions.Db)
		tools.Composer(&toolsDefVersions.Composer)
		tools.NodeJs(&toolsDefVersions.NodeJs)
		tools.Yarn(&toolsDefVersions.Yarn)

		tools.Redis(&toolsDefVersions.Redis)
		tools.RabbitMQ(&toolsDefVersions.RabbitMQ)
		tools.Hosts(projectName, &toolsDefVersions.Hosts, projectConf)

		projects.SetEnvForProject(projectName, toolsDefVersions, configs.GetProjectConfigOnly(projectName))
		paths.MakeDirsByPath(paths.GetExecDirPath() + "/projects/" + projectName + "/backup/db")

		fmtc.SuccessLn("\n" + "Finish set up environment")
		fmtc.ToDoLn("Optionally, you can configure SSH access to the development server in order ")
		fmtc.ToDoLn("to synchronize the database and media files. Enter SSH data in ")
		fmtc.ToDoLn(paths.GetExecDirPath() + "/projects/" + projectName + "/env.txt")

		docker.Down(withVolumes)
		start.Execute(withChown, projectConf)
	}
}
