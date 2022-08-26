package commands

import (
	"log"

	"github.com/faradey/madock/src/docker/builder"
)

func DB(flag, option string) {
	if flag == "import" {
		builder.DbImport(option)
	} else if flag == "export" {
		builder.DbExport()
	} else if flag == "soft-clean" {
		builder.DbSoftClean()
	} else if flag == "info" {
		builder.DbInfo()
	} else {
		log.Fatal("The specified parameters were not found.")
	}
}
