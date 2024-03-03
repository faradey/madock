package logs

import (
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/docker"
	"github.com/faradey/madock/src/helper/logger"
	"os"
	"os/exec"
)

type ArgsStruct struct {
	attr.Arguments
	Service string `arg:"-s,--service" help:"Service name (php, nginx, db, etc.)"`
}

func Execute() {
	args := attr.Parse(new(ArgsStruct)).(*ArgsStruct)

	service := "php"
	projectConf := configs.GetCurrentProjectConfig()
	if projectConf["platform"] == "pwa" {
		service = "nodejs"
	}

	if args.Service != "" {
		service = args.Service
	}

	projectName := configs.GetProjectName()
	cmd := exec.Command("docker", "logs", docker.GetContainerName(projectConf, projectName, service))
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		logger.Fatal(err)
	}
}
