package commands

import (
	"github.com/faradey/madock/src/cli/fmtc"
	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/docker/builder"
	"log"
)

func Start() {
	if !configs.IsHasNotConfig() {
		fmtc.SuccessLn("Start containers in detached mode")
		builder.Up()
		fmtc.SuccessLn("Done")
	} else {
		fmtc.WarningLn("Set up the project")
		fmtc.ToDoLn("Run madock setup")
	}
}

func Stop(flag string) {
	if !configs.IsHasNotConfig() {
		if flag == "--all" {
			builder.DownAll()
		} else {
			builder.Down()
		}
		fmtc.SuccessLn("Done")
	} else {
		fmtc.WarningLn("Set up the project")
		fmtc.ToDoLn("Run madock setup")
	}
}

func Magento(flag string) {
	builder.Magento(flag)
}

func Composer(flag string) {
	builder.Composer(flag)
}

func DB(flag, option string) {
	if flag == "--import" {
		builder.DbImport(option)
	} else if flag == "--export" {
		builder.DbExport()
	} else {
		log.Fatal("The specified parameters were not found.")
	}
}

func IsNotDefine() {
	fmtc.ErrorLn("The command is not defined. Run 'madock help' to invoke help")
}
