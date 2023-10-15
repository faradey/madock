package commands

import (
	"github.com/faradey/madock/src/cli/attr"
	"github.com/faradey/madock/src/cli/fmtc"
	"github.com/faradey/madock/src/configs"
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
	}
	if len(paths.GetActiveProjects()) == 0 {
		Proxy("stop")
	}
}

func Restart() {
	Stop()
	Start()
}

func Rebuild() {
	if !configs.IsHasNotConfig() {
		fmtc.SuccessLn("Stop containers")
		builder.Down(false)
		if len(paths.GetActiveProjects()) == 0 {
			Proxy("stop")
		}
		fmtc.SuccessLn("Start containers in detached mode")
		builder.UpWithBuild()
		fmtc.SuccessLn("Done")
	} else {
		fmtc.WarningLn("Set up the project")
		fmtc.ToDoLn("Run madock setup")
	}
}
