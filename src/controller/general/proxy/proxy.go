package proxy

import (
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	configs2 "github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/docker"
)

type ArgsStruct struct {
	attr.Arguments
	Force bool `arg:"-f,--force" help:"Force"`
}

func Execute(flag string) {
	args := attr.Parse(new(ArgsStruct)).(*ArgsStruct)

	if !configs2.IsHasNotConfig() {
		projectConf := configs2.GetCurrentProjectConfig()
		if projectConf["proxy/enabled"] == "true" {
			if flag == "prune" {
				docker.DownNginx(args.Force)
			} else if flag == "stop" {
				docker.StopNginx(args.Force)
			} else if flag == "restart" {
				docker.StopNginx(args.Force)
				docker.UpNginx()
			} else if flag == "start" {
				docker.UpNginx()
			} else if flag == "rebuild" {
				docker.DownNginx(args.Force)
				docker.UpNginxWithBuild(true)
			}
			fmtc.SuccessLn("Done")
		} else {
			fmtc.WarningLn("Proxy service is disabled. Run 'madock service:enable proxy' to enable it")
		}
	} else {
		fmtc.WarningLn("Set up the project")
		fmtc.ToDoLn("Run madock setup")
	}
}
