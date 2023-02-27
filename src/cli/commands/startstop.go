package commands

import (
	"github.com/faradey/madock/src/cli/attr"
	"github.com/faradey/madock/src/cli/fmtc"
	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/docker/builder"
)

func Start() {
	if !configs.IsHasNotConfig() {
		fmtc.SuccessLn("Start containers in detached mode")
		builder.Start(attr.Options.WithChown)
		fmtc.SuccessLn("Done")
	} else {
		fmtc.WarningLn("Set up the project")
		fmtc.ToDoLn("Run madock setup")
	}
}

func Stop() {
	builder.Stop()
	if len(builder.GetActiveProjects()) == 0 {
		Proxy("stop")
	}
}

func Restart() {
	Stop()
	Start()
}

func Rebuild() {
	if !configs.IsHasNotConfig() {
		fmtc.SuccessLn("Stop containers")
		builder.Down(false)
		if len(builder.GetActiveProjects()) == 0 {
			Proxy("stop")
		}
		fmtc.SuccessLn("Start containers in detached mode")
		builder.UpWithBuild()
		fmtc.SuccessLn("Done")
	} else {
		fmtc.WarningLn("Set up the project")
		fmtc.ToDoLn("Run madock setup")
	}
}
