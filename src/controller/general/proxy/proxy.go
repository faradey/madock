package proxy

import (
	"os"

	"github.com/faradey/madock/v3/src/command"
	"github.com/faradey/madock/v3/src/helper/cli/arg_struct"
	"github.com/faradey/madock/v3/src/helper/cli/attr"
	"github.com/faradey/madock/v3/src/helper/cli/fmtc"
	configs2 "github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/docker"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"proxy:start"},
		Handler:  func() { Execute("start") },
		Help:     "Start proxy",
		Category: "proxy",
		ArgsType: new(arg_struct.ControllerGeneralProxy),
	})
	command.Register(&command.Definition{
		Aliases:  []string{"proxy:stop"},
		Handler:  func() { Execute("stop") },
		Help:     "Stop proxy",
		Category: "proxy",
		ArgsType: new(arg_struct.ControllerGeneralProxy),
	})
	command.Register(&command.Definition{
		Aliases:  []string{"proxy:restart"},
		Handler:  func() { Execute("restart") },
		Help:     "Restart proxy",
		Category: "proxy",
		ArgsType: new(arg_struct.ControllerGeneralProxy),
	})
	command.Register(&command.Definition{
		Aliases:  []string{"proxy:rebuild"},
		Handler:  func() { Execute("rebuild") },
		Help:     "Rebuild proxy",
		Category: "proxy",
		ArgsType: new(arg_struct.ControllerGeneralProxy),
	})
	command.Register(&command.Definition{
		Aliases:  []string{"proxy:reload"},
		Handler:  func() { Execute("reload") },
		Help:     "Reload proxy config",
		Category: "proxy",
		ArgsType: new(arg_struct.ControllerGeneralProxy),
	})
	command.Register(&command.Definition{
		Aliases:  []string{"proxy:prune"},
		Handler:  func() { Execute("prune") },
		Help:     "Prune proxy",
		Category: "proxy",
		ArgsType: new(arg_struct.ControllerGeneralProxy),
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
				// reload is the one verb that cannot create what it needs: it
				// execs `nginx -s reload` inside a container that has to
				// already be running. When it is not, docker fails and the
				// configuration on disk is simply not applied — printing the
				// success line below would report the intent instead of what
				// happened, and the caller (a script, or a person who has just
				// changed a host) would carry on believing the proxy is
				// serving the new config.
				if err := docker.ReloadNginx(); err != nil {
					fmtc.ErrorLn("The proxy configuration was not reloaded: the proxy is not running")
					fmtc.ToDoLn("Run madock proxy:start")
					os.Exit(1)
				}
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
