package setup

import (
	"fmt"
	"strings"

	"github.com/faradey/madock/src/controller/general/rebuild"
	"github.com/faradey/madock/src/helper/cli/arg_struct"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/configs/projects"
	"github.com/faradey/madock/src/helper/paths"
	"github.com/faradey/madock/src/helper/setup/tools"
	"github.com/faradey/madock/src/model/versions"
	"github.com/faradey/madock/src/model/versions/custom"
	"github.com/faradey/madock/src/model/versions/languages"
)

func Execute(projectName string, projectConf map[string]string, continueSetup bool, args *arg_struct.ControllerGeneralSetup, language string) {
	toolsDefVersions := custom.GetVersions()
	toolsDefVersions.Language = language

	if continueSetup {
		fmt.Println("")

		switch language {
		case "php":
			setupPhpTools(&toolsDefVersions, args)
		case "nodejs":
			setupNodeJsTools(&toolsDefVersions, args)
		case "python":
			setupPythonTools(&toolsDefVersions, args)
		case "golang":
			setupGolangTools(&toolsDefVersions, args)
		case "ruby":
			setupRubyTools(&toolsDefVersions, args)
		case "none":
			// No language-specific interactive selectors needed
		}

		// Common services (DB, search, redis, etc.)
		if args.Db == "" {
			tools.Db(&toolsDefVersions.Db)
		} else {
			toolsDefVersions.Db = args.Db
		}

		if language == "php" {
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

func setupPhpTools(toolsDefVersions *versions.ToolsVersions, args *arg_struct.ControllerGeneralSetup) {
	if args.Php == "" {
		tools.Php(&toolsDefVersions.Php)
	} else {
		toolsDefVersions.Php = args.Php
	}
	if args.Composer == "" {
		tools.Composer(&toolsDefVersions.Composer)
	} else {
		toolsDefVersions.Composer = args.Composer
	}
}

func setupNodeJsTools(toolsDefVersions *versions.ToolsVersions, args *arg_struct.ControllerGeneralSetup) {
	defVersions := languages.GetDefaultVersions("nodejs")
	toolsDefVersions.NodeJs = defVersions["nodejs/version"]
	if args.NodeJs == "" {
		tools.NodeJs(&toolsDefVersions.NodeJs)
	} else {
		toolsDefVersions.NodeJs = args.NodeJs
	}
	toolsDefVersions.Yarn = defVersions["nodejs/yarn/version"]
	if args.Yarn == "" {
		tools.Yarn(&toolsDefVersions.Yarn)
	} else {
		toolsDefVersions.Yarn = args.Yarn
	}
}

func setupPythonTools(toolsDefVersions *versions.ToolsVersions, _ *arg_struct.ControllerGeneralSetup) {
	defVersions := languages.GetDefaultVersions("python")
	toolsDefVersions.Python = defVersions["python/version"]
	tools.PythonVersion(&toolsDefVersions.Python)
}

func setupGolangTools(toolsDefVersions *versions.ToolsVersions, _ *arg_struct.ControllerGeneralSetup) {
	defVersions := languages.GetDefaultVersions("golang")
	toolsDefVersions.Golang = defVersions["go/version"]
	tools.GoVersion(&toolsDefVersions.Golang)
}

func setupRubyTools(toolsDefVersions *versions.ToolsVersions, _ *arg_struct.ControllerGeneralSetup) {
	defVersions := languages.GetDefaultVersions("ruby")
	toolsDefVersions.Ruby = defVersions["ruby/version"]
	tools.RubyVersion(&toolsDefVersions.Ruby)
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
