package start

import (
	"fmt"
	"time"

	"github.com/faradey/madock/v3/src/command"
	"github.com/faradey/madock/v3/src/controller/platform"
	"github.com/faradey/madock/v3/src/helper/cli/arg_struct"
	"github.com/faradey/madock/v3/src/helper/cli/attr"
	"github.com/faradey/madock/v3/src/helper/cli/fmtc"
	configs2 "github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/docker"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"start"},
		Handler:  Execute,
		Help:     "Start containers",
		Category: "general",
		ArgsType: new(arg_struct.ControllerGeneralStart),
	})
}

func Execute() {
	args := attr.Parse(new(arg_struct.ControllerGeneralStart)).(*arg_struct.ControllerGeneralStart)

	if configs2.IsHasConfig("") {
		projectName := configs2.GetProjectName()
		projectConf := configs2.GetProjectConfig(projectName)
		platformName := projectConf["platform"]
		startTime := time.Now()

		fmtc.TitleLn("Starting containers...")

		handler := platform.GetOrDefault(platformName)
		handler.Start(projectName, args.WithChown, projectConf)

		elapsed := time.Since(startTime).Round(time.Second)
		fmt.Println("")

		// "up" returning zero only says the containers were created. Report a
		// service whose main process is already gone instead of claiming
		// success and leaving the reason in a log nobody has been told to read.
		if dead := docker.NotRunning(projectName); len(dead) > 0 {
			fmtc.WarningIconLn(fmt.Sprintf("Containers started in %s, but some are not running:", elapsed))
			for _, service := range dead {
				line := "  " + service.Service + " — " + service.State
				if service.ExitCode != 0 {
					line += fmt.Sprintf(" (exit %d)", service.ExitCode)
				}
				fmtc.WarningLn(line)
			}
			fmtc.ToDoLn("madock logs -s <service>")
			return
		}

		fmtc.SuccessIconLn(fmt.Sprintf("Containers started in %s", elapsed))
	} else {
		fmtc.WarningIconLn("Set up the project")
		fmtc.ToDoLn("Run madock setup")
	}
}
