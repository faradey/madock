package proxy

import (
	"github.com/faradey/madock/src/helper/cli/arg_struct"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	configs2 "github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/docker"
)

func Execute(flag string) {
	args := attr.Parse(new(arg_struct.ControllerGeneralProxy)).(*arg_struct.ControllerGeneralProxy)

	if configs2.IsHasConfig("") {
		projectName := configs2.GetProjectName()
		projectConf := configs2.GetCurrentProjectConfig()
		if projectConf["proxy/enabled"] == "true" {
			if flag == "prune" {
				docker.DownNginx(args.Force)
			} else if flag == "stop" {
				docker.StopNginx(args.Force)
			} else if flag == "restart" {
				docker.StopNginx(args.Force)
				docker.UpNginx(projectName)
			} else if flag == "start" {
				docker.UpNginx(projectName)
			} else if flag == "rebuild" {
				docker.DownNginx(args.Force)
				docker.UpNginxWithBuild(projectName, true)
			} else if flag == "reload" {
				docker.ReloadNginx()
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
