package logs

import (
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/jessevdk/go-flags"
	"log"
	"os"
	"os/exec"
)

type ArgsStruct struct {
	attr.Arguments
	Service string `long:"service" short:"s" description:"Service name"`
}

func Execute() {
	args := getArgs()

	service := "php"
	projectConf := configs.GetCurrentProjectConfig()
	if projectConf["PLATFORM"] == "pwa" {
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
