package cron

import (
	"github.com/faradey/madock/v3/src/command"
	"github.com/faradey/madock/v3/src/helper/cli/attr"
	"github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/docker"
)

type ArgsStruct struct {
	attr.Arguments
}

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"cron:enable"},
		Handler:  Enable,
		Help:     "Enable cron",
		Category: "cron",
	})
	command.Register(&command.Definition{
		Aliases:  []string{"cron:disable"},
		Handler:  Disable,
		Help:     "Disable cron",
		Category: "cron",
	})
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
