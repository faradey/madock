package spree

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
		Aliases:  []string{"spree"},
		Handler:  Execute,
		Help:     "Execute Spree rails CLI (e.g. `madock spree console`)",
		Category: "spree",
	})
}

func Execute() {
	flag := cliHelper.NormalizeCliCommandWithJoin(os.Args[2:])
	projectName := configs.GetProjectName()
	projectConf := configs.GetCurrentProjectConfig()

	if projectConf["platform"] != "spree" {
		fmtc.Warning("This command is not supported for " + projectConf["platform"])
		return
	}

	workdir := projectConf["workdir"]
	if workdir == "" {
		workdir = "/var/www/html"
	}

	// Spree (Rails) reads config from process env. Source .env before
	// invoking rails so DATABASE_URL / REDIS_URL / SECRET_KEY_BASE are
	// available to commands like `console`, `db:migrate`, `routes`.
	loadEnv := "set -a; [ -f .env ] && . ./.env; set +a"

	cmd := "cd " + workdir + " && " + loadEnv + " && bundle exec rails " + flag

	err := docker.ContainerExec(
		docker.GetContainerName(projectConf, projectName, "ruby"),
		"ruby",
		true,
		"bash", "-c",
		cmd,
	)
	if err != nil {
		logger.Fatal(err)
	}
}
