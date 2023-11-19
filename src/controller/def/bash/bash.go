package bash

import (
	"github.com/faradey/madock/src/cli/attr"
	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/service"
	"github.com/jessevdk/go-flags"
	"log"
	"os"
)

type ArgsStruct struct {
	attr.Arguments
	Service string `long:"service" short:"s" description:"Service name"`
	User    string `long:"user" short:"u" description:"User"`
}

func Bash() {
	args := getArgs()

	containerName := "php"
	user := "root"
	projectConf := configs.GetCurrentProjectConfig()
	if projectConf["PLATFORM"] == "pwa" {
		containerName = "nodejs"
	}

	if args.Service != "" {
		containerName = args.Service
	}

	if args.User != "" {
		user = args.User
	}

	service.Bash(containerName, user)
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
