package rebuild

import (
	"github.com/faradey/madock/src/controller/general/proxy"
	"github.com/faradey/madock/src/helper/cli/arg_struct"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/docker"
	"github.com/faradey/madock/src/helper/logger"
	"github.com/faradey/madock/src/helper/paths"
	"os"
)

func Execute() {
	args := attr.Parse(new(arg_struct.ControllerGeneralRebuild)).(*arg_struct.ControllerGeneralRebuild)

	if configs.IsHasConfig("") {
		projectName := configs.GetProjectName()
		if paths.IsFileExist(paths.GetExecDirPath() + "/cache/conf-cache") {
			err := os.Remove(paths.GetExecDirPath() + "/cache/conf-cache")
			if err != nil {
				logger.Fatal(err)
			}
		}
		fmtc.SuccessLn("Stop containers")
		if args.Force {
			docker.Kill(projectName)
		} else {
			docker.Down(projectName, false)
		}
		if len(paths.GetActiveProjects()) == 0 {
			proxy.Execute("stop")
		}
		fmtc.SuccessLn("Start containers in detached mode")
		docker.UpWithBuild(projectName, args.WithChown)
		fmtc.SuccessLn("Done")
	} else {
		fmtc.WarningLn("Set up the project")
		fmtc.ToDoLn("Run madock setup")
	}
}
