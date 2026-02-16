package web

import (
	"github.com/faradey/madock/src/command"
	"github.com/faradey/madock/src/helper/cli"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/docker"
	"github.com/faradey/madock/src/helper/logger"
	"os"
	"os/exec"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"shopify:web", "sy:w"},
		Handler:  Execute,
		Help:     "Execute Shopify web",
		Category: "shopify",
	})
}

func Execute() {
	flag := cli.NormalizeCliCommandWithJoin(os.Args[2:])
	projectName := configs.GetProjectName()
	projectConf := configs.GetCurrentProjectConfig()
	service, user, workdir := cli.GetEnvForUserServiceWorkdir("php", "www-data", projectConf["workdir"])
	cmd := exec.Command("docker", "exec", "-it", "-u", user, docker.GetContainerName(projectConf, projectName, service), "bash", "-c", "cd "+workdir+"/web && "+flag)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		logger.Fatal(err)
	}
}
