package patch

import (
	"github.com/faradey/madock/src/helper/cli"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/docker"
	"log"
	"os"
	"os/exec"
)

type ArgsStruct struct {
	attr.Arguments
	File  string `arg:"--file" help:"File path"`
	Name  string `arg:"-n,--name" help:"Parameter name"`
	Title string `arg:"-t,--title" help:"Title"`
	Force bool   `arg:"-f,--force" help:"Force"`
}

func Create() {
	args := attr.Parse(new(ArgsStruct)).(*ArgsStruct)

	filePath := args.File
	patchName := args.Name
	title := args.Title
	force := args.Force

	if filePath == "" {
		log.Fatal("The --file option is incorrect or not specified.")
	}

	projectName := configs.GetProjectName()
	projectConf := configs.GetCurrentProjectConfig()
	isForce := ""
	if force {
		isForce = "f"
	}
	service, user, workdir := cli.GetEnvForUserServiceWorkdir("php", "www-data", projectConf["workdir"])
	cmd := exec.Command("docker", "exec", "-it", "-u", user, docker.GetContainerName(projectConf, projectName, service), "php", "/var/www/scripts/php/patch-create.php", workdir, filePath, patchName, title, isForce)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}
