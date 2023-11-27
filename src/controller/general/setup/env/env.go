package env

import (
	"encoding/json"
	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/docker/scripts"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/paths"
	"github.com/jessevdk/go-flags"
	"log"
	"os"
)

type ArgsStruct struct {
	attr.Arguments
	Force bool   `long:"force" short:"f" description:"Force"`
	Host  string `long:"host" short:"h" description:"Host"`
}

func Execute() {
	args := getArgs()

	envFile := paths.GetRunDirPath() + "/app/etc/env.php"
	if _, err := os.Stat(envFile); !os.IsNotExist(err) && !args.Force {
		log.Fatal("The env.php file is already exist.")
	} else {
		data, err := json.Marshal(configs.GetCurrentProjectConfig())
		if err != nil {
			log.Fatal(err)
		}
		scripts.CreateEnv(string(data), args.Host)
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
