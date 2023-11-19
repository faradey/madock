package setup

import (
	"fmt"
	"github.com/faradey/madock/src/cli/attr"
	"github.com/faradey/madock/src/cli/fmtc"
	"github.com/faradey/madock/src/configs/projects"
	"github.com/faradey/madock/src/docker/builder"
	"github.com/faradey/madock/src/paths"
	"github.com/faradey/madock/src/versions/shopify"
)

func Shopify(projectName string, projectConf map[string]string, continueSetup bool) {
	toolsDefVersions := shopify.GetVersions()

	if continueSetup {
		fmt.Println("")

		Php(&toolsDefVersions.Php)
		Db(&toolsDefVersions.Db)
		Composer(&toolsDefVersions.Composer)
		NodeJs(&toolsDefVersions.NodeJs)
		Yarn(&toolsDefVersions.Yarn)

		Redis(&toolsDefVersions.Redis)
		RabbitMQ(&toolsDefVersions.RabbitMQ)
		Hosts(projectName, &toolsDefVersions.Hosts, projectConf)

		projects.SetEnvForProject(projectName, toolsDefVersions, projectConf)
		paths.MakeDirsByPath(paths.GetExecDirPath() + "/projects/" + projectName + "/backup/db")

		fmtc.SuccessLn("\n" + "Finish set up environment")
		fmtc.ToDoLn("Optionally, you can configure SSH access to the development server in order ")
		fmtc.ToDoLn("to synchronize the database and media files. Enter SSH data in ")
		fmtc.ToDoLn(paths.GetExecDirPath() + "/projects/" + projectName + "/env.txt")
	}

	builder.Down(attr.Options.WithVolumes)
	builder.StartShopify(attr.Options.WithChown, projectConf)
}
