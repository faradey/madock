package rebuild

import (
	"fmt"
	"os"
	"time"

	"github.com/faradey/madock/src/controller/general/proxy"
	"github.com/faradey/madock/src/helper/cli/arg_struct"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/docker"
	"github.com/faradey/madock/src/helper/logger"
	"github.com/faradey/madock/src/helper/paths"
)

func Execute() {
	args := attr.Parse(new(arg_struct.ControllerGeneralRebuild)).(*arg_struct.ControllerGeneralRebuild)

	if configs.IsHasConfig("") {
		projectName := configs.GetProjectName()
		startTime := time.Now()

		// Clear config cache with spinner
		spinner := fmtc.NewSpinner("Preparing environment...")
		spinner.Start()
		if paths.IsFileExist(paths.GetExecDirPath() + "/cache/conf-cache") {
			err := os.Remove(paths.GetExecDirPath() + "/cache/conf-cache")
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
		fmtc.SuccessIconLn(fmt.Sprintf("Rebuild completed in %s", elapsed))
	} else {
		fmtc.WarningIconLn("Set up the project")
		fmtc.ToDoLn("Run madock setup")
	}
}
