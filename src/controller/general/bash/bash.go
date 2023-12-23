package bash

import (
	"github.com/alexflint/go-arg"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/docker"
	"log"
	"os"
	"os/exec"
)

type ArgsStruct struct {
	attr.Arguments
	Service string `arg:"-s,--service" help:"Service name (php, nginx, db, etc.)"`
	User    string `arg:"-u,--user" help:"User"`
}

func Bash() {
	args := getArgs()

	service := "php"
	user := "root"
	projectConf := configs.GetCurrentProjectConfig()
	if projectConf["PLATFORM"] == "pwa" {
		service = "nodejs"
	}

	if args.Service != "" {
		service = args.Service
	}

	if args.User != "" {
		user = args.User
	}

	projectName := configs.GetProjectName()
	cmd := exec.Command("docker", "exec", "-it", "-u", user, docker.GetContainerName(projectConf, projectName, service), "bash")
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
	if attr.IsParseArgs && len(os.Args) > 2 {
		argsOrigin := os.Args[2:]
		p, err := arg.NewParser(arg.Config{
			IgnoreEnv: true,
		}, args)

		if err != nil {
			log.Fatal(err)
		}

		err = p.Parse(argsOrigin)

		if err != nil {
			log.Fatal(err)
		}
	}

	attr.IsParseArgs = false
	return args
}
