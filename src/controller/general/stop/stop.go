package stop

import (
	"github.com/faradey/madock/v4/src/command"
	"github.com/faradey/madock/v4/src/controller/general/proxy"
	"github.com/faradey/madock/v4/src/controller/platform"
	"github.com/faradey/madock/v4/src/helper/cli/fmtc"
	"github.com/faradey/madock/v4/src/helper/configs"
	"github.com/faradey/madock/v4/src/helper/docker"
	"github.com/faradey/madock/v4/src/helper/paths"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"stop"},
		Handler:  Execute,
		Help:     "Stop containers",
		Category: "general",
	})
}

func Execute() {
	projectName := configs.GetProjectName()

	// Stopping must not depend on the config parsing. `handler.Stop` reads the
	// compose file, not the project config, so a project whose `.madock/config.xml`
	// is broken can still be taken down — which is what matters when the broken
	// config is why it needs stopping. The platform only chooses the handler, and
	// every handler's Stop is the same `docker compose stop`; an unreadable config
	// falls back to the default handler and says so, instead of exiting 1 with the
	// containers left running.
	platformName := ""
	parseErr := configs.ProjectConfigParses(projectName)
	if parseErr != nil {
		fmtc.WarningIconLn("Stopping \"" + projectName + "\" with its config.xml unreadable (" + parseErr.Error() + "): the containers are stopped anyway.")
	} else {
		platformName = configs.GetCurrentProjectConfig()["platform"]
	}

	handler := platform.GetOrDefault(platformName)
	handler.Stop(projectName)

	if len(paths.GetActiveProjects()) == 0 {
		if parseErr != nil {
			// proxy.Execute reads this project's config to see whether it uses
			// the proxy, and on an unreadable one ended the process after the
			// containers were already down: `stop` of the last running project
			// exited 1. With no project left running the shared proxy serves
			// nobody, so it is stopped without asking.
			docker.StopNginx(false)
			return
		}
		proxy.Execute("stop")
	}
}
