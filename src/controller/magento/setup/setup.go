package setup

import (
	"fmt"
	"github.com/faradey/madock/src/controller/general/install"
	"github.com/faradey/madock/src/controller/general/rebuild"
	"github.com/faradey/madock/src/helper/cli"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	configs2 "github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/configs/projects"
	"github.com/faradey/madock/src/helper/docker"
	"github.com/faradey/madock/src/helper/logger"
	"github.com/faradey/madock/src/helper/paths"
	"github.com/faradey/madock/src/helper/setup/tools"
	"github.com/faradey/madock/src/model/versions/magento2"
	"os"
	"os/exec"
	"strings"
)

func Execute(projectName string, projectConf map[string]string, continueSetup, doDownload, doInstall, isSampleData bool) {
	toolsDefVersions := magento2.GetVersions("")

	mageVersion := ""
	if toolsDefVersions.Php == "" {
		fmt.Println("")
		fmtc.Title("Specify Magento version: ")
		mageVersion, _ = tools.Waiter()
		if mageVersion != "" {
			toolsDefVersions = magento2.GetVersions(mageVersion)
		} else {
			Execute(projectName, projectConf, continueSetup, doDownload, doInstall, isSampleData)
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
		} else if toolsDefVersions.SearchEngine == "OpenSearch" {
			tools.OpenSearch(&toolsDefVersions.OpenSearch)
		}

		tools.Redis(&toolsDefVersions.Redis)
		tools.RabbitMQ(&toolsDefVersions.RabbitMQ)
		tools.Hosts(projectName, &toolsDefVersions.Hosts, projectConf)

		projects.SetEnvForProject(projectName, toolsDefVersions, configs2.GetProjectConfigOnly(projectName))
		paths.MakeDirsByPath(paths.GetExecDirPath() + "/projects/" + projectName + "/backup/db")

		fmtc.SuccessLn("\n" + "Finish set up environment")
		fmtc.ToDoLn("Optionally, you can configure SSH access to the development server in order ")
		fmtc.ToDoLn("to synchronize the database and media files. Enter SSH data in ")
		fmtc.ToDoLn(paths.GetExecDirPath() + "/projects/" + projectName + "/config.xml")

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

	if doDownload || doInstall || continueSetup {
		rebuild.Execute()
	}

	if doDownload {
		DownloadMagento(projectName, edition, mageVersion, isSampleData)
	}

	if doInstall {
		install.Magento(projectName, toolsDefVersions.Magento)
	}
}

func DownloadMagento(projectName, edition, version string, isSampleData bool) {
	projectConf := configs2.GetCurrentProjectConfig()
	sampleData := ""
	if isSampleData {
		sampleData = " && bin/magento sampledata:deploy"
	}
	service, user, workdir := cli.GetEnvForUserServiceWorkdir("php", "www-data", projectConf["workdir"])
	command := []string{
		"exec",
		"-it",
		"-u",
		user,
		docker.GetContainerName(projectConf, projectName, service),
		"bash",
		"-c",
		"cd " + workdir + " " +
			"&& rm -r -f " + workdir + "/download-magento123456789 " +
			"&& mkdir " + workdir + "/download-magento123456789 " +
			"&& composer create-project --repository-url=https://repo.magento.com/ magento/project-" + edition + "-edition:" + version + " ./download-magento123456789 " +
			"&& shopt -s dotglob " +
			"&& mv  -v ./download-magento123456789/* ./ " +
			"&& rm -r -f ./download-magento123456789 " +
			"&& composer install" + sampleData,
	}
	cmd := exec.Command("docker", command...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		logger.Fatal(err)
	}
}
