package setup

import (
	"fmt"
	"github.com/faradey/madock/src/cli/attr"
	"github.com/faradey/madock/src/cli/fmtc"
	"github.com/faradey/madock/src/configs/projects"
	"github.com/faradey/madock/src/controller/general/install"
	"github.com/faradey/madock/src/docker/builder"
	"github.com/faradey/madock/src/paths"
	"github.com/faradey/madock/src/versions/magento2"
	"strings"
)

func Magento2(projectName string, projectConf map[string]string, continueSetup bool) {
	toolsDefVersions := magento2.GetVersions("")

	mageVersion := ""
	if toolsDefVersions.Php == "" {
		fmt.Println("")
		fmtc.Title("Specify Magento version: ")
		mageVersion, _ = Waiter()
		if mageVersion != "" {
			toolsDefVersions = magento2.GetVersions(mageVersion)
		} else {
			Magento2(projectName, projectConf, continueSetup)
			return
		}
	}

	edition := "community"

	if continueSetup {
		fmt.Println("")
		fmtc.Title("Your Magento version is " + toolsDefVersions.Magento)

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
		Hosts(projectName, &toolsDefVersions.Hosts, projectConf)

		projects.SetEnvForProject(projectName, toolsDefVersions, projectConf)
		paths.MakeDirsByPath(paths.GetExecDirPath() + "/projects/" + projectName + "/backup/db")

		fmtc.SuccessLn("\n" + "Finish set up environment")
		fmtc.ToDoLn("Optionally, you can configure SSH access to the development server in order ")
		fmtc.ToDoLn("to synchronize the database and media files. Enter SSH data in ")
		fmtc.ToDoLn(paths.GetExecDirPath() + "/projects/" + projectName + "/env.txt")

		if attr.Options.Download {
			fmt.Println("")
			fmtc.TitleLn("Specify Magento version: ")
			fmt.Println("1) Community (default)")
			fmt.Println("2) Enterprise")
			edition, _ = Waiter()
			edition = strings.TrimSpace(edition)
			if edition != "1" && edition != "2" && edition != "" {
				fmtc.ErrorLn("The specified edition '" + edition + "' is incorrect.")
				return
			}
			if edition == "1" || edition == "" {
				edition = "community"
			} else if edition == "2" {
				edition = "enterprise"
			}
		}
	}

	builder.Down(attr.Options.WithVolumes)
	builder.StartMagento2(attr.Options.WithChown, projectConf)

	if attr.Options.Download {
		DownloadMagento(projectName, mageVersion, edition)
	}

	if attr.Options.Install {
		install.Magento(projectName, toolsDefVersions.Magento)
	}
}

func DownloadMagento(projectName, mageVersion, edition string) {
	builder.DownloadMagento(projectName, edition, mageVersion)
}
