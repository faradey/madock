package bash

import (
	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/jessevdk/go-flags"
	"log"
	"os"
	"os/exec"
	"strings"
)

type ArgsStruct struct {
	attr.Arguments
	Service string `long:"service" short:"s" description:"Service name"`
	User    string `long:"user" short:"u" description:"User"`
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
	cmd := exec.Command("docker", "exec", "-it", "-u", user, strings.ToLower(projectConf["CONTAINER_NAME_PREFIX"])+strings.ToLower(projectName)+"-"+service+"-1", "bash")
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
