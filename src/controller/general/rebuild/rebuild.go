package rebuild

import (
	"fmt"
	"os"
	"time"

	"github.com/faradey/madock/v3/src/command"
	"github.com/faradey/madock/v3/src/controller/general/proxy"
	"github.com/faradey/madock/v3/src/helper/cli/arg_struct"
	"github.com/faradey/madock/v3/src/helper/cli/attr"
	"github.com/faradey/madock/v3/src/helper/cli/fmtc"
	"github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/docker"
	"github.com/faradey/madock/v3/src/helper/logger"
	"github.com/faradey/madock/v3/src/helper/paths"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"rebuild"},
		Handler:  Execute,
		Help:     "Rebuild containers",
		Category: "general",
		ArgsType: new(arg_struct.ControllerGeneralRebuild),
	})
}

func Execute() {
	args := attr.Parse(new(arg_struct.ControllerGeneralRebuild)).(*arg_struct.ControllerGeneralRebuild)

	if configs.IsHasConfig("") {
		projectName := configs.GetProjectName()
		startTime := time.Now()

		// Clear config cache with spinner
		spinner := fmtc.NewSpinner("Preparing environment...")
		spinner.Start()
		if paths.IsFileExist(paths.CacheDir() + "/conf-cache") {
			err := os.Remove(paths.CacheDir() + "/conf-cache")
			if err != nil {
				spinner.StopWithError("Failed to clear cache")
				logger.Fatal(err)
			}
		}
		spinner.StopWithSuccess("Environment prepared")

		// Stop containers
		fmt.Println("")
		fmtc.TitleLn("Stopping containers...")
		if args.Force {
			docker.Kill(projectName)
		} else {
			docker.Down(projectName, false)
		}
		if len(paths.GetActiveProjects()) == 0 {
			proxy.Execute("stop")
		}

		// Start containers
		fmt.Println("")
		fmtc.TitleLn("Starting containers...")
		docker.UpWithBuild(projectName, args.WithChown)

		// Done
		elapsed := time.Since(startTime).Round(time.Second)
		fmt.Println("")
		// Same honesty as `start`: creating containers is not the same as having
		// them run, and a rebuild is the moment a newly enabled service shows
		// whether it can. Seen with an image built for another architecture —
		// the container was created, started, died on exec, and the rebuild
		// reported success with nothing to suggest looking further.
		// A moment to settle first. A container that cannot exec its entrypoint
		// is reported as "Started" and is gone a breath later, so asking
		// immediately after `up` sees it running and reports success — which is
		// exactly the answer this check exists to stop giving. Two seconds
		// against a rebuild that takes tens of them.
		time.Sleep(2 * time.Second)

		if dead := docker.NotRunning(projectName); len(dead) > 0 {
			fmtc.WarningIconLn(fmt.Sprintf("Rebuild completed in %s, but some services are not running:", elapsed))
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

		fmtc.SuccessIconLn(fmt.Sprintf("Rebuild completed in %s", elapsed))
	} else {
		fmtc.WarningIconLn("Set up the project")
		fmtc.ToDoLn("Run madock setup")
	}
}
