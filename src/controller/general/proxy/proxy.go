package proxy

import (
	"github.com/faradey/madock/src/command"
	"github.com/faradey/madock/src/helper/cli/arg_struct"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	configs2 "github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/docker"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"proxy:start"},
		Handler:  func() { Execute("start") },
		Help:     "Start proxy",
		Category: "proxy",
	})
	command.Register(&command.Definition{
		Aliases:  []string{"proxy:stop"},
		Handler:  func() { Execute("stop") },
		Help:     "Stop proxy",
		Category: "proxy",
	})
	command.Register(&command.Definition{
		Aliases:  []string{"proxy:restart"},
		Handler:  func() { Execute("restart") },
		Help:     "Restart proxy",
		Category: "proxy",
	})
	command.Register(&command.Definition{
		Aliases:  []string{"proxy:rebuild"},
		Handler:  func() { Execute("rebuild") },
		Help:     "Rebuild proxy",
		Category: "proxy",
	})
	command.Register(&command.Definition{
		Aliases:  []string{"proxy:reload"},
		Handler:  func() { Execute("reload") },
		Help:     "Reload proxy config",
		Category: "proxy",
	})
	command.Register(&command.Definition{
		Aliases:  []string{"proxy:prune"},
		Handler:  func() { Execute("prune") },
		Help:     "Prune proxy",
		Category: "proxy",
	})
}

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
