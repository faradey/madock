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
		projectConfig := configs.GetCurrentProjectConfig()
		fmtc.SuccessLn("Start containers in detached mode")
		if projectConfig["PLATFORM"] == "magento2" {
			builder.StartMagento2(attr.Options.WithChown, projectConfig)
		} else if projectConfig["PLATFORM"] == "pwa" {
			builder.StartPWA(attr.Options.WithChown)
		} else if projectConfig["PLATFORM"] == "shopify" {
			builder.StartShopify(attr.Options.WithChown, projectConfig)
		} else if projectConfig["PLATFORM"] == "custom" {
			builder.StartCustom(attr.Options.WithChown, projectConfig)
		}
		fmtc.SuccessLn("Done")
	} else {
		fmtc.WarningLn("Set up the project")
		fmtc.ToDoLn("Run madock setup")
	}
}

func Stop() {
	projectConfig := configs.GetCurrentProjectConfig()
	if projectConfig["PLATFORM"] == "magento2" {
		builder.StopMagento2()
	} else if projectConfig["PLATFORM"] == "pwa" {
		builder.StopPWA()
	} else if projectConfig["PLATFORM"] == "shopify" {
		builder.StopShopify()
	} else if projectConfig["PLATFORM"] == "custom" {
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
