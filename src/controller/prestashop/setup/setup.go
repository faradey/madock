package setup

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/faradey/madock/v3/src/controller/general/install"
	"github.com/faradey/madock/v3/src/controller/general/rebuild"
	"github.com/faradey/madock/v3/src/helper/cli"
	"github.com/faradey/madock/v3/src/helper/cli/arg_struct"
	"github.com/faradey/madock/v3/src/helper/cli/fmtc"
	configs2 "github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/configs/projects"
	"github.com/faradey/madock/v3/src/helper/docker"
	"github.com/faradey/madock/v3/src/helper/logger"
	"github.com/faradey/madock/v3/src/helper/paths"
	"github.com/faradey/madock/v3/src/helper/setup/tools"
	"github.com/faradey/madock/v3/src/model/versions/prestashop"
	setupreg "github.com/faradey/madock/v3/src/setup"
)

type Handler struct{}

func (h *Handler) Execute(ctx *setupreg.SetupContext) {
	Execute(ctx.ProjectName, ctx.ProjectConf, ctx.ContinueSetup, ctx.Args)
}

func init() {
	setupreg.Register(setupreg.PlatformInfo{
		Name:        "prestashop",
		DisplayName: "PrestaShop",
		Language:    "php",
		Order:       50,
	}, &Handler{})
}

func Execute(projectName string, projectConf map[string]string, continueSetup bool, args *arg_struct.ControllerGeneralSetup) {
	toolsDefVersions := prestashop.GetVersions("")

	platformVersion := ""
	if args.PlatformVersion != "" {
		platformVersion = args.PlatformVersion
		if args.Php != "" {
			toolsDefVersions.Php = args.Php
		}
	}

	if args.Download && continueSetup {
		if platformVersion == "" {
			fmt.Println("")
			fmtc.Title("Specify PrestaShop version: ")
			platformVersion, _ = tools.Waiter()
		}
		if platformVersion != "" {
			toolsDefVersions = prestashop.GetVersions(platformVersion)
		} else {
			Execute(projectName, projectConf, continueSetup, args)
			return
		}
	}

	if continueSetup {
		fmt.Println("")
		fmtc.Title("Your PrestaShop version is " + toolsDefVersions.PlatformVersion)

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
			if args.SearchEngineVersion == "" {
				tools.Elastic(&toolsDefVersions.Elastic)
			} else {
				toolsDefVersions.Elastic = args.SearchEngineVersion
			}
		} else if toolsDefVersions.SearchEngine == "OpenSearch" {
			if args.SearchEngineVersion == "" {
				tools.OpenSearch(&toolsDefVersions.OpenSearch)
			} else {
				toolsDefVersions.OpenSearch = args.SearchEngineVersion
			}
		}

		if args.Redis == "" {
			tools.Redis(&toolsDefVersions.Redis)
		} else {
			toolsDefVersions.Redis = args.Redis
		}

		if args.Valkey == "" {
			tools.Valkey(&toolsDefVersions.Valkey)
		} else {
			toolsDefVersions.Valkey = args.Valkey
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
		DownloadPrestashop(projectName, platformVersion)
	}

	if args.Install {
		install.PrestaShop(projectName, toolsDefVersions.PlatformVersion, args.SampleData)
	}
}

func DownloadPrestashop(projectName, version string) {
	projectConf := configs2.GetCurrentProjectConfig()
	service, user, workdir := cli.GetEnvForUserServiceWorkdir("php", "www-data", projectConf["workdir"])
	ttyFlag := "-i"
	if docker.IsTTYAvailable() {
		ttyFlag = "-it"
	}
	command := []string{
		"exec",
		ttyFlag,
		"-u",
		user,
		docker.GetContainerName(projectConf, projectName, service),
		"bash",
		"-c",
		"cd " + workdir + " " +
			"&& rm -r -f " + workdir + "/download-presta123456789 " +
			"&& mkdir " + workdir + "/download-presta123456789 " +
			"&& wget -P ./download-presta123456789 https://github.com/PrestaShop/PrestaShop/releases/download/" + version + "/prestashop_" + version + ".zip " +
			"&& unzip ./download-presta123456789/prestashop_" + version + ".zip -d ./download-presta123456789/" +
			"&& rm ./download-presta123456789/prestashop_" + version + ".zip " +
			"&& unzip ./download-presta123456789/prestashop.zip " +
			"&& rm -r -f ./download-presta123456789 ",
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
