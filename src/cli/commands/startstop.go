package commands

import (
	"github.com/faradey/madock/src/cli/attr"
	"github.com/faradey/madock/src/cli/fmtc"
	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/controller/general/proxy"
	"github.com/faradey/madock/src/docker/builder"
	"github.com/faradey/madock/src/paths"
)

func Start() {
	if !configs.IsHasNotConfig() {
		projectConf := configs.GetCurrentProjectConfig()
		fmtc.SuccessLn("Start containers in detached mode")
		if projectConf["PLATFORM"] == "magento2" {
			builder.StartMagento2(attr.Options.WithChown, projectConf)
		} else if projectConf["PLATFORM"] == "pwa" {
			builder.StartPWA(attr.Options.WithChown)
		} else if projectConf["PLATFORM"] == "shopify" {
			builder.StartShopify(attr.Options.WithChown, projectConf)
		} else if projectConf["PLATFORM"] == "custom" {
			builder.StartCustom(attr.Options.WithChown, projectConf)
		}
		fmtc.SuccessLn("Done")
	} else {
		fmtc.WarningLn("Set up the project")
		fmtc.ToDoLn("Run madock setup")
	}
}

func Stop() {
	projectConf := configs.GetCurrentProjectConfig()
	if projectConf["PLATFORM"] == "magento2" {
		builder.StopMagento2()
	} else if projectConf["PLATFORM"] == "pwa" {
		builder.StopPWA()
	} else if projectConf["PLATFORM"] == "shopify" {
		builder.StopShopify()
	} else if projectConf["PLATFORM"] == "custom" {
		builder.StopCustom()
	}
	if len(paths.GetActiveProjects()) == 0 {
		proxy.Execute("stop")
	}
}

func Restart() {
	Stop()
	Start()
}
