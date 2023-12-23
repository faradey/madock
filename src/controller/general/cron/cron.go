package cron

import (
	"github.com/alexflint/go-arg"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/docker"
	"log"
	"os"
)

type ArgsStruct struct {
	attr.Arguments
}

func Enable() {
	getArgs()
	docker.CronExecute(true, true)
}

func Disable() {
	getArgs()
	docker.CronExecute(false, true)
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
