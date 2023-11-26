package setup

import (
	"fmt"
	"github.com/faradey/madock/src/cli/fmtc"
	"github.com/faradey/madock/src/configs/projects"
	"github.com/faradey/madock/src/controller/general/install"
	"github.com/faradey/madock/src/docker/builder"
	"github.com/faradey/madock/src/helper/paths"
	"github.com/faradey/madock/src/helper/setup/tools"
	"github.com/faradey/madock/src/versions/magento2"
	"strings"
)

func Execute(projectName string, projectConf map[string]string, continueSetup, doDownload, doInstall, withChown, withVolumes, isSampleData bool) {
	toolsDefVersions := magento2.GetVersions("")

	mageVersion := ""
	if toolsDefVersions.Php == "" {
		fmt.Println("")
		fmtc.Title("Specify Magento version: ")
		mageVersion, _ = tools.Waiter()
		if mageVersion != "" {
			toolsDefVersions = magento2.GetVersions(mageVersion)
		} else {
			Execute(projectName, projectConf, continueSetup, doDownload, doInstall, withChown, withVolumes, isSampleData)
			return
		}
	}

	edition := "community"

	if continueSetup {
		fmt.Println("")
		fmtc.Title("Your Magento version is " + toolsDefVersions.Magento)

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
		tools.Hosts(projectName, &toolsDefVersions.Hosts, projectConf)

		projects.SetEnvForProject(projectName, toolsDefVersions, projectConf)
		paths.MakeDirsByPath(paths.GetExecDirPath() + "/projects/" + projectName + "/backup/db")

		fmtc.SuccessLn("\n" + "Finish set up environment")
		fmtc.ToDoLn("Optionally, you can configure SSH access to the development server in order ")
		fmtc.ToDoLn("to synchronize the database and media files. Enter SSH data in ")
		fmtc.ToDoLn(paths.GetExecDirPath() + "/projects/" + projectName + "/env.txt")

		if doDownload {
			fmt.Println("")
			fmtc.TitleLn("Specify Magento version: ")
			fmt.Println("1) Community (default)")
			fmt.Println("2) Enterprise")
			edition, _ = tools.Waiter()
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

	if doDownload || doInstall {
		builder.Down(withVolumes)
		builder.StartMagento2(withChown, projectConf)
	}

	if doDownload {
		builder.DownloadMagento(projectName, edition, mageVersion, isSampleData)
	}

	if doInstall {
		install.Magento(projectName, toolsDefVersions.Magento)
	}
}
