package start

import (
	startCustom "github.com/faradey/madock/src/controller/custom/start"
	startMagento2 "github.com/faradey/madock/src/controller/magento/start"
	startPwa "github.com/faradey/madock/src/controller/pwa/start"
	builder2 "github.com/faradey/madock/src/controller/shopify/start"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	configs2 "github.com/faradey/madock/src/helper/configs"
	"github.com/jessevdk/go-flags"
	"log"
	"os"
)

type ArgsStruct struct {
	attr.Arguments
	WithChown bool `long:"with-chown" short:"c" description:"With Chown"`
}

func Execute() {
	args := getArgs()

	if !configs2.IsHasNotConfig() {
		projectConf := configs2.GetCurrentProjectConfig()
		fmtc.SuccessLn("Start containers in detached mode")
		if projectConf["PLATFORM"] == "magento2" {
			startMagento2.Execute(args.WithChown, projectConf)
		} else if projectConf["PLATFORM"] == "pwa" {
			startPwa.Execute(args.WithChown)
		} else if projectConf["PLATFORM"] == "shopify" {
			builder2.Execute(args.WithChown, projectConf)
		} else if projectConf["PLATFORM"] == "custom" {
			startCustom.Execute(args.WithChown, projectConf)
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
