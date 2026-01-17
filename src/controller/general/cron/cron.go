package cron

import (
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/docker"
)

type ArgsStruct struct {
	attr.Arguments
}

func Enable() {
	attr.Parse(new(ArgsStruct))
	projectName := configs.GetProjectName()
	projectConfig := configs.GetProjectConfig(projectName)
	configs.SetParam(projectName, "cron/enabled", "true", projectConfig["activeScope"], "")
	docker.CronExecute(projectName, true, true)
}

func Disable() {
	attr.Parse(new(ArgsStruct))
	projectName := configs.GetProjectName()
	projectConfig := configs.GetProjectConfig(projectName)
	configs.SetParam(projectName, "cron/enabled", "false", projectConfig["activeScope"], "")
	docker.CronExecute(projectName, false, true)
}
