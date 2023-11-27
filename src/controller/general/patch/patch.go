package patch

import (
	"github.com/faradey/madock/src/helper/cli"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/jessevdk/go-flags"
	"log"
	"os"
	"os/exec"
)

type ArgsStruct struct {
	attr.Arguments
	File  string `long:"file" description:"File path"`
	Name  string `long:"name" short:"n" description:"Parameter name"`
	Title string `long:"title" short:"t" description:"Title"`
	Force bool   `long:"force" short:"f" description:"Force"`
}

func Create() {
	args := getArgs()

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
	service, user, workdir := cli.GetEnvForUserServiceWorkdir("php", "www-data", projectConf["WORKDIR"])
	cmd := exec.Command("docker", "exec", "-it", "-u", user, docker.GetContainerName(projectConf, projectName, service), "php", "/var/www/scripts/php/patch-create.php", workdir, filePath, patchName, title, isForce)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func getArgs() *ArgsStruct {
	args := new(ArgsStruct)
	if len(os.Args) > 2 {
		argsOrigin := os.Args[2:]
		var err error
		_, err = flags.ParseArgs(args, argsOrigin)

		if err != nil {
			log.Fatal(err)
		}
	}

	return args
}
