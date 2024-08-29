package setup

import (
	"fmt"
	"github.com/faradey/madock/src/controller/general/install"
	"github.com/faradey/madock/src/controller/general/rebuild"
	"github.com/faradey/madock/src/helper/cli"
	"github.com/faradey/madock/src/helper/cli/arg_struct"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	configs2 "github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/configs/projects"
	"github.com/faradey/madock/src/helper/docker"
	"github.com/faradey/madock/src/helper/logger"
	"github.com/faradey/madock/src/helper/paths"
	"github.com/faradey/madock/src/helper/setup/tools"
	"github.com/faradey/madock/src/model/versions/shopware"
	"os"
	"os/exec"
)

func Execute(projectName string, projectConf map[string]string, continueSetup bool, args *arg_struct.ControllerGeneralSetup) {
	toolsDefVersions := shopware.GetVersions("")

	platformVersion := ""
	if args.PlatformVersion != "" {
		platformVersion = args.PlatformVersion
		if args.Php != "" {
			toolsDefVersions.Php = args.Php
		}
	}

	if continueSetup {
		if toolsDefVersions.Php == "" {
			if platformVersion == "" {
				fmt.Println("")
				fmtc.Title("Specify Shopware version: ")
				platformVersion, _ = tools.Waiter()
			}
			if platformVersion != "" {
				toolsDefVersions = shopware.GetVersions(platformVersion)
			} else {
				Execute(projectName, projectConf, continueSetup, args)
				return
			}
		}

		fmt.Println("")
		fmtc.Title("Your Shopware version is " + toolsDefVersions.PlatformVersion)

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
			tools.Hosts(projectName, &toolsDefVersions.Hosts, projectConf)
		} else {
			toolsDefVersions.Hosts = args.Hosts
		}

		projects.SetEnvForProject(projectName, toolsDefVersions, configs2.GetProjectConfigOnly(projectName))
		paths.MakeDirsByPath(paths.GetExecDirPath() + "/projects/" + projectName + "/backup/db")

		fmtc.SuccessLn("\n" + "Finish set up environment")
		fmtc.ToDoLn("Optionally, you can configure SSH access to the development server in order ")
		fmtc.ToDoLn("to synchronize the database and media files. Enter SSH data in ")
		fmtc.ToDoLn(paths.GetExecDirPath() + "/projects/" + projectName + "/config.xml")
	}

	if args.Download || args.Install || continueSetup {
		rebuild.Execute()
	}

	if args.Download {
		DownloadShopware(projectName, platformVersion, args.SampleData)
	}

	if args.Install {
		install.Shopware(projectName, toolsDefVersions.PlatformVersion, args.SampleData)
	}
}

func DownloadShopware(projectName, version string, isSampleData bool) {
	projectConf := configs2.GetCurrentProjectConfig()
	sampleData := ""
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
			"&& rm -r -f " + workdir + "/download-shopware123456789 " +
			"&& mkdir " + workdir + "/download-shopware123456789 " +
			"&& composer create-project shopware/production:" + version + " ./download-shopware123456789 " +
			"&& shopt -s dotglob " +
			"&& mv  -v ./download-shopware123456789/* ./ " +
			"&& rm -r -f ./download-shopware123456789 " +
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
