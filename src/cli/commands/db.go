package commands

import (
	"github.com/faradey/madock/src/docker/builder"
)

func DBImport() {
	builder.DbImport()
}

func DBExport() {
	builder.DbExport()
}

func DBInfo() {
	builder.DbInfo()
}
