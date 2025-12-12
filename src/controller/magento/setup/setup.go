package setup

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

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
	"github.com/faradey/madock/src/model/versions/magento2"
)

func Execute(projectName string, projectConf map[string]string, continueSetup bool, args *arg_struct.ControllerGeneralSetup) {
	toolsDefVersions := magento2.GetVersions("")

	mageVersion := ""
	if args.PlatformVersion != "" {
		mageVersion = args.PlatformVersion
		if args.Php != "" {
			toolsDefVersions.Php = args.Php
		}
	}

	// Initialize progress tracker for setup steps
	setupSteps := []string{
		"Magento Version",
		"PHP Version",
		"Database Version",
		"Composer Version",
		"Search Engine",
		"Search Engine Version",
		"Redis Version",
		"Valkey Version",
		"RabbitMQ Version",
		"Hosts Configuration",
	}
	tools.InitProgress(setupSteps)
	currentStep := 1

	if toolsDefVersions.Php == "" {
		if mageVersion == "" {
			tools.SetProgressStep(currentStep)
			fmtc.Title("Specify Magento version: ")
			mageVersion, _ = tools.Waiter()
		}
		if mageVersion != "" {
			toolsDefVersions = magento2.GetVersions(mageVersion)
		} else {
			Execute(projectName, projectConf, continueSetup, args)
			return
		}
	}
	currentStep++

	edition := "community"
	if args.PlatformEdition != "" {
		edition = args.PlatformEdition
	}

	if continueSetup {
		fmt.Println("")
		fmtc.Title("Your Magento version is " + toolsDefVersions.PlatformVersion)

		// Step 2: PHP Version
		tools.SetProgressStep(currentStep)
		if args.Php == "" {
			tools.Php(&toolsDefVersions.Php)
		} else {
			toolsDefVersions.Php = args.Php
		}
		currentStep++

		// Step 3: Database Version
		tools.SetProgressStep(currentStep)
		if args.Db == "" {
			tools.Db(&toolsDefVersions.Db)
		} else {
			toolsDefVersions.Db = args.Db
		}
		currentStep++

		// Step 4: Composer Version
		tools.SetProgressStep(currentStep)
		if args.Composer == "" {
			tools.Composer(&toolsDefVersions.Composer)
		} else {
			toolsDefVersions.Composer = args.Composer
		}
		currentStep++

		// Step 5: Search Engine
		tools.SetProgressStep(currentStep)
		if args.SearchEngine == "" {
			tools.SearchEngine(&toolsDefVersions.SearchEngine)
		} else {
			toolsDefVersions.SearchEngine = args.SearchEngine
		}
		currentStep++

		// Step 6: Search Engine Version
		tools.SetProgressStep(currentStep)
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
		currentStep++

		// Step 7: Redis Version
		tools.SetProgressStep(currentStep)
		if args.Redis == "" {
			tools.Redis(&toolsDefVersions.Redis)
		} else {
			toolsDefVersions.Redis = args.Redis
		}
		currentStep++

		// Step 8: Valkey Version
		tools.SetProgressStep(currentStep)
		if args.Valkey == "" {
			tools.Valkey(&toolsDefVersions.Valkey)
		} else {
			toolsDefVersions.Valkey = args.Valkey
		}
		currentStep++

		// Step 9: RabbitMQ Version
		tools.SetProgressStep(currentStep)
		if args.RabbitMQ == "" {
			tools.RabbitMQ(&toolsDefVersions.RabbitMQ)
		} else {
			toolsDefVersions.RabbitMQ = args.RabbitMQ
		}
		currentStep++

		// Step 10: Hosts Configuration
		tools.SetProgressStep(currentStep)
		if args.Hosts == "" {
			tools.Hosts(projectName, &toolsDefVersions.Hosts, projectConf)
		} else {
			toolsDefVersions.Hosts = args.Hosts
		}

		projects.SetEnvForProject(projectName, toolsDefVersions, configs2.GetProjectConfigOnly(projectName))
		paths.MakeDirsByPath(paths.GetExecDirPath() + "/projects/" + projectName + "/backup/db")

		// Show completion
		tools.CompleteProgress()
		fmt.Println("")
		fmtc.ToDoLn("Optionally, you can configure SSH access to the development server in order ")
		fmtc.ToDoLn("to synchronize the database and media files. Enter SSH data in ")
		fmtc.ToDoLn(paths.GetExecDirPath() + "/projects/" + projectName + "/config.xml")

		if args.Download && args.PlatformEdition == "" {
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

	if args.Download || args.Install || continueSetup {
		rebuild.Execute()
	}

	if args.Download {
		DownloadMagento(projectName, edition, mageVersion, args.SampleData)
	}

	if args.Install {
		install.Magento(projectName, toolsDefVersions.PlatformVersion)
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
