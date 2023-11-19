package proxy

import (
	"github.com/faradey/madock/src/cli/fmtc"
	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/docker/builder"
)

func Execute(flag string) {
	if !configs.IsHasNotConfig() {
		projectConfig := configs.GetCurrentProjectConfig()
		if projectConfig["PROXY_ENABLED"] == "true" {
			if flag == "prune" {
				builder.DownNginx()
			} else if flag == "stop" {
				builder.StopNginx()
			} else if flag == "restart" {
				builder.StopNginx()
				builder.UpNginx()
			} else if flag == "start" {
				builder.UpNginx()
			} else if flag == "rebuild" {
				builder.DownNginx()
				builder.UpNginx()
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
