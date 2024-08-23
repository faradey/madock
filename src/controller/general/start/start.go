package start

import (
	startCustom "github.com/faradey/madock/src/controller/custom/start"
	startMagento2 "github.com/faradey/madock/src/controller/magento/start"
	startPwa "github.com/faradey/madock/src/controller/pwa/start"
	builder2 "github.com/faradey/madock/src/controller/shopify/start"
	startShopware "github.com/faradey/madock/src/controller/shopware/start"
	"github.com/faradey/madock/src/helper/cli/arg_struct"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	configs2 "github.com/faradey/madock/src/helper/configs"
)

func Execute() {
	args := attr.Parse(new(arg_struct.ControllerGeneralStart)).(*arg_struct.ControllerGeneralStart)

	if configs2.IsHasConfig("") {
		projectName := configs2.GetProjectName()
		projectConf := configs2.GetProjectConfig(projectName)
		platform := projectConf["platform"]
		fmtc.SuccessLn("Start containers in detached mode")
		if platform == "magento2" {
			startMagento2.Execute(projectName, args.WithChown, projectConf)
		} else if platform == "pwa" {
			startPwa.Execute(projectName, args.WithChown)
		} else if platform == "shopify" {
			builder2.Execute(projectName, args.WithChown, projectConf)
		} else if platform == "custom" {
			startCustom.Execute(projectName, args.WithChown, projectConf)
		} else if platform == "shopware" {
			startShopware.Execute(projectName, args.WithChown, projectConf)
		}
		fmtc.SuccessLn("Done")
	} else {
		fmtc.WarningLn("Set up the project")
		fmtc.ToDoLn("Run madock setup")
	}
}
