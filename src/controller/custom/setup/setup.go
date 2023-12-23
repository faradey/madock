package setup

import (
	"fmt"
	"github.com/faradey/madock/src/controller/custom/start"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/configs/projects"
	"github.com/faradey/madock/src/helper/docker"
	"github.com/faradey/madock/src/helper/paths"
	"github.com/faradey/madock/src/helper/setup/tools"
	"github.com/faradey/madock/src/model/versions/custom"
	"strings"
)

func Execute(projectName string, projectConf map[string]string, continueSetup bool) {
	toolsDefVersions := custom.GetVersions()

	if continueSetup {
		fmt.Println("")

		tools.Php(&toolsDefVersions.Php)
		tools.Db(&toolsDefVersions.Db)
		tools.Composer(&toolsDefVersions.Composer)
		tools.SearchEngine(&toolsDefVersions.SearchEngine)
		if toolsDefVersions.SearchEngine == "Elasticsearch" {
			tools.Elastic(&toolsDefVersions.Elastic)
		} else {
			tools.OpenSearch(&toolsDefVersions.OpenSearch)
		}

		tools.Redis(&toolsDefVersions.Redis)
		tools.RabbitMQ(&toolsDefVersions.RabbitMQ)
		hostsCustom(projectName, &toolsDefVersions.Hosts, projectConf)

		projects.SetEnvForProject(projectName, toolsDefVersions, configs.GetProjectConfigOnly(projectName))
		paths.MakeDirsByPath(paths.GetExecDirPath() + "/projects/" + projectName + "/backup/db")

		fmtc.SuccessLn("\n" + "Finish set up environment")
		fmtc.ToDoLn("Optionally, you can configure SSH access to the development server in order ")
		fmtc.ToDoLn("to synchronize the database and media files. Enter SSH data in ")
		fmtc.ToDoLn(paths.GetExecDirPath() + "/projects/" + projectName + "/env.txt")

		docker.Down(false)
		start.Execute(false, projectConf)
	}
}

func hostsCustom(projectName string, defVersion *string, projectConf map[string]string) {
	host := strings.ToLower(projectName + projectConf["DEFAULT_HOST_FIRST_LEVEL"])
	if val, ok := projectConf["HOSTS"]; ok && val != "" {
		host = val
	}
	fmtc.TitleLn("Hosts")
	fmt.Println("Input format: a.example.com b.example.com")
	fmt.Println("Recommended host: " + host)
	*defVersion = host
	availableVersions := []string{"Custom", projectName + projectConf["DEFAULT_HOST_FIRST_LEVEL"], "loc." + projectName + ".com"}
	tools.PrepareVersions(availableVersions)
	tools.Invitation(defVersion)
	tools.WaiterAndProceed(defVersion, availableVersions)
}
