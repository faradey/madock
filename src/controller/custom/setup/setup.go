package setup

import (
	"fmt"
	"github.com/faradey/madock/src/controller/general/rebuild"
	"github.com/faradey/madock/src/helper/cli/arg_struct"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/configs/projects"
	"github.com/faradey/madock/src/helper/paths"
	"github.com/faradey/madock/src/helper/setup/tools"
	"github.com/faradey/madock/src/model/versions/custom"
	"strings"
)

func Execute(projectName string, projectConf map[string]string, continueSetup bool, args *arg_struct.ControllerGeneralSetup) {
	toolsDefVersions := custom.GetVersions()

	if continueSetup {
		fmt.Println("")

		if args.Php == "" {
			tools.Php(&toolsDefVersions.Php)
		} else {
			toolsDefVersions.Php = args.Php
		}
		if args.Db == "" {
			tools.Db(&toolsDefVersions.Db)
		} else {
			toolsDefVersions.Db = args.Db
		}
		if args.Composer == "" {
			tools.Composer(&toolsDefVersions.Composer)
		} else {
			toolsDefVersions.Composer = args.Composer
		}
		if args.SearchEngine == "" {
			tools.SearchEngine(&toolsDefVersions.SearchEngine)
		} else {
			toolsDefVersions.SearchEngine = args.SearchEngine
		}
		if toolsDefVersions.SearchEngine == "Elasticsearch" {
			if args.Elastic == "" {
				tools.Elastic(&toolsDefVersions.Elastic)
			} else {
				toolsDefVersions.Elastic = args.Elastic
			}
		} else if toolsDefVersions.SearchEngine == "OpenSearch" {
			if args.OpenSearch == "" {
				tools.OpenSearch(&toolsDefVersions.OpenSearch)
			} else {
				toolsDefVersions.OpenSearch = args.OpenSearch
			}
		}

		if args.Redis == "" {
			tools.Redis(&toolsDefVersions.Redis)
		} else {
			toolsDefVersions.Redis = args.Redis
		}
		if args.RabbitMQ == "" {
			tools.RabbitMQ(&toolsDefVersions.RabbitMQ)
		} else {
			toolsDefVersions.RabbitMQ = args.RabbitMQ
		}
		if args.Hosts == "" {
			hostsCustom(projectName, &toolsDefVersions.Hosts, projectConf)
		} else {
			toolsDefVersions.Hosts = args.Hosts
		}
		projects.SetEnvForProject(projectName, toolsDefVersions, configs.GetProjectConfigOnly(projectName))
		paths.MakeDirsByPath(paths.GetExecDirPath() + "/projects/" + projectName + "/backup/db")

		fmtc.SuccessLn("\n" + "Finish set up environment")
		fmtc.ToDoLn("Optionally, you can configure SSH access to the development server in order ")
		fmtc.ToDoLn("to synchronize the database and media files. Enter SSH data in ")
		fmtc.ToDoLn(paths.GetExecDirPath() + "/projects/" + projectName + "/config.xml")

		rebuild.Execute()
	}
}

func hostsCustom(projectName string, defVersion *string, projectConf map[string]string) {
	host := strings.ToLower(projectName + projectConf["nginx/default_host_first_level"])
	hosts := configs.GetHosts(projectConf)
	if len(hosts) > 0 {
		var hostItems []string
		for _, hostItem := range hosts {
			hostItems = append(hostItems, hostItem["name"])
		}
		host = strings.Join(hostItems, " ")
	}
	fmtc.TitleLn("Hosts")
	fmt.Println("Input format: a.example.com b.example.com")
	fmt.Println("Recommended host: " + host)
	*defVersion = host
	availableVersions := []string{"Custom", projectName + projectConf["nginx/default_host_first_level"], "loc." + projectName + ".com"}
	tools.PrepareVersions(availableVersions)
	tools.Invitation(defVersion)
	tools.WaiterAndProceed(defVersion, availableVersions)
}
