package start

import (
	startCustom "github.com/faradey/madock/src/controller/custom/start"
	startMagento2 "github.com/faradey/madock/src/controller/magento/start"
	startPwa "github.com/faradey/madock/src/controller/pwa/start"
	builder2 "github.com/faradey/madock/src/controller/shopify/start"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	configs2 "github.com/faradey/madock/src/helper/configs"
)

type ArgsStruct struct {
	attr.Arguments
	WithChown bool `arg:"-c,--with-chown" help:"With Chown"`
}

func Execute() {
	args := attr.Parse(new(ArgsStruct)).(*ArgsStruct)

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
