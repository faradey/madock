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
	"github.com/faradey/madock/src/model/versions/shopify"
	setupreg "github.com/faradey/madock/src/setup"
)

type Handler struct{}

func (h *Handler) Execute(ctx *setupreg.SetupContext) {
	Execute(ctx.ProjectName, ctx.ProjectConf, ctx.ContinueSetup, ctx.Args)
}

func init() {
	setupreg.Register(setupreg.PlatformInfo{
		Name:     "shopify",
		Language: "php",
		Order:    30,
	}, &Handler{})
}

func Execute(projectName string, projectConf map[string]string, continueSetup bool, args *arg_struct.ControllerGeneralSetup) {
	toolsDefVersions := shopify.GetVersions()

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
		if args.NodeJs == "" {
			tools.NodeJs(&toolsDefVersions.NodeJs)
		} else {
			toolsDefVersions.NodeJs = args.NodeJs
		}
		if args.Yarn == "" {
			tools.Yarn(&toolsDefVersions.Yarn)
		} else {
			toolsDefVersions.Yarn = args.Yarn
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

		projects.SetEnvForProject(projectName, toolsDefVersions, configs.GetProjectConfigOnly(projectName))
		paths.MakeDirsByPath(paths.GetExecDirPath() + "/projects/" + projectName + "/backup/db")

		fmtc.SuccessLn("\n" + "Finish set up environment")
		fmtc.ToDoLn("Optionally, you can configure SSH access to the development server in order ")
		fmtc.ToDoLn("to synchronize the database and media files. Enter SSH data in ")
		fmtc.ToDoLn(paths.GetExecDirPath() + "/projects/" + projectName + "/config.xml")

		rebuild.Execute()
	}
}
