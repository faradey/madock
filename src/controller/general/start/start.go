package start

import (
	"github.com/faradey/madock/src/cli/attr"
	"github.com/faradey/madock/src/cli/fmtc"
	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/docker/builder"
	"github.com/jessevdk/go-flags"
	"log"
	"os"
)

type ArgsStruct struct {
	attr.Arguments
	WithChown bool `long:"with-chown" description:"With Chown"`
}

func Execute() {
	args := getArgs()

	if !configs.IsHasNotConfig() {
		projectConf := configs.GetCurrentProjectConfig()
		fmtc.SuccessLn("Start containers in detached mode")
		if projectConf["PLATFORM"] == "magento2" {
			builder.StartMagento2(args.WithChown, projectConf)
		} else if projectConf["PLATFORM"] == "pwa" {
			builder.StartPWA(args.WithChown)
		} else if projectConf["PLATFORM"] == "shopify" {
			builder.StartShopify(args.WithChown, projectConf)
		} else if projectConf["PLATFORM"] == "custom" {
			builder.StartCustom(args.WithChown, projectConf)
		}
		fmtc.SuccessLn("Done")
	} else {
		fmtc.WarningLn("Set up the project")
		fmtc.ToDoLn("Run madock setup")
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
