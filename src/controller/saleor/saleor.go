package saleor

import (
	"os"

	"github.com/faradey/madock/v3/src/command"
	cliHelper "github.com/faradey/madock/v3/src/helper/cli"
	"github.com/faradey/madock/v3/src/helper/cli/fmtc"
	"github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/docker"
	"github.com/faradey/madock/v3/src/helper/logger"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"saleor"},
		Handler:  Execute,
		Help:     "Execute Saleor manage.py CLI",
		Category: "saleor",
	})
}

func Execute() {
	flag := cliHelper.NormalizeCliCommandWithJoin(os.Args[2:])
	projectName := configs.GetProjectName()
	projectConf := configs.GetCurrentProjectConfig()

	if projectConf["platform"] != "saleor" {
		fmtc.Warning("This command is not supported for " + projectConf["platform"])
		return
	}

	workdir := projectConf["workdir"]
	if workdir == "" {
		workdir = "/var/www/html"
	}

	// Prefer `uv run` when a uv.lock is present; otherwise plain python.
	cmd := "cd " + workdir + " && if [ -f uv.lock ] && command -v uv >/dev/null 2>&1; then uv run python manage.py " + flag + "; else python manage.py " + flag + "; fi"

	err := docker.ContainerExec(
		docker.GetContainerName(projectConf, projectName, "python"),
		"saleor",
		true,
		"bash", "-c",
		cmd,
	)
	if err != nil {
		logger.Fatal(err)
	}
}
