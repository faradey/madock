package commands

import (
	"github.com/faradey/madock/src/cli/fmtc"
	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/docker/builder"
)

func Start() {
	if !configs.IsHasNotConfig() {
		fmtc.SuccessLn("Start containers in detached mode")
		builder.Start()
		fmtc.SuccessLn("Done")
	} else {
		fmtc.WarningLn("Set up the project")
		fmtc.ToDoLn("Run madock setup")
	}
}

func Stop() {
	builder.Stop()
}

func Restart() {
	Stop()
	Start()
}

func Rebuild() {
	if !configs.IsHasNotConfig() {
		fmtc.SuccessLn("Stop containers")
		builder.Down()
		fmtc.SuccessLn("Start containers in detached mode")
		builder.UpWithBuild()
		fmtc.SuccessLn("Done")
	} else {
		fmtc.WarningLn("Set up the project")
		fmtc.ToDoLn("Run madock setup")
	}
}
