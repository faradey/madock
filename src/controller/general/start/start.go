package start

import (
	"github.com/alexflint/go-arg"
	startCustom "github.com/faradey/madock/src/controller/custom/start"
	startMagento2 "github.com/faradey/madock/src/controller/magento/start"
	startPwa "github.com/faradey/madock/src/controller/pwa/start"
	builder2 "github.com/faradey/madock/src/controller/shopify/start"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	configs2 "github.com/faradey/madock/src/helper/configs"
	"log"
	"os"
)

type ArgsStruct struct {
	attr.Arguments
	WithChown bool `arg:"-c,--with-chown" help:"With Chown"`
}

func Execute() {
	args := getArgs()

	if !configs2.IsHasNotConfig() {
		projectConf := configs2.GetCurrentProjectConfig()
		fmtc.SuccessLn("Start containers in detached mode")
		if projectConf["platform"] == "magento2" {
			startMagento2.Execute(args.WithChown, projectConf)
		} else if projectConf["platform"] == "pwa" {
			startPwa.Execute(args.WithChown)
		} else if projectConf["platform"] == "shopify" {
			builder2.Execute(args.WithChown, projectConf)
		} else if projectConf["platform"] == "custom" {
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
