package setup

import (
	"fmt"
	"github.com/faradey/madock/src/cli/attr"
	"github.com/faradey/madock/src/cli/fmtc"
	"github.com/faradey/madock/src/configs/projects"
	"github.com/faradey/madock/src/docker/builder"
	"github.com/faradey/madock/src/paths"
	"github.com/faradey/madock/src/versions/custom"
	"strings"
)

func Custom(projectName string, projectConf map[string]string, continueSetup bool) {
	toolsDefVersions := custom.GetVersions()

	if continueSetup {
		fmt.Println("")

		Php(&toolsDefVersions.Php)
		Db(&toolsDefVersions.Db)
		Composer(&toolsDefVersions.Composer)
		SearchEngine(&toolsDefVersions.SearchEngine)
		if toolsDefVersions.SearchEngine == "Elasticsearch" {
			Elastic(&toolsDefVersions.Elastic)
		} else {
			OpenSearch(&toolsDefVersions.OpenSearch)
		}

		Redis(&toolsDefVersions.Redis)
		RabbitMQ(&toolsDefVersions.RabbitMQ)
		HostsCustom(projectName, &toolsDefVersions.Hosts, projectConf)

		projects.SetEnvForProject(projectName, toolsDefVersions, projectConf)
		paths.MakeDirsByPath(paths.GetExecDirPath() + "/projects/" + projectName + "/backup/db")

		fmtc.SuccessLn("\n" + "Finish set up environment")
		fmtc.ToDoLn("Optionally, you can configure SSH access to the development server in order ")
		fmtc.ToDoLn("to synchronize the database and media files. Enter SSH data in ")
		fmtc.ToDoLn(paths.GetExecDirPath() + "/projects/" + projectName + "/env.txt")
	}

	builder.Down(attr.Options.WithVolumes)
	builder.StartCustom(attr.Options.WithChown, projectConf)
}

func HostsCustom(projectName string, defVersion *string, projectConf map[string]string) {
	host := strings.ToLower(projectName + projectConf["DEFAULT_HOST_FIRST_LEVEL"])
	if val, ok := projectConf["HOSTS"]; ok && val != "" {
		host = val
	}
	fmtc.TitleLn("Hosts")
	fmt.Println("Input format: a.example.com b.example.com")
	fmt.Println("Recommended host: " + host)
	*defVersion = host
	availableVersions := []string{"Custom", projectName + projectConf["DEFAULT_HOST_FIRST_LEVEL"], "loc." + projectName + ".com"}
	prepareVersions(availableVersions)
	invitation(defVersion)
	waiterAndProceed(defVersion, availableVersions)
}
