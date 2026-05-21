package saleor

import (
	"github.com/faradey/madock/v3/src/model/versions"
)

func init() {
	versions.RegisterProvider("saleor", func(_ string) versions.ToolsVersions {
		return GetVersions()
	})
}

func GetVersions() versions.ToolsVersions {
	return versions.ToolsVersions{
		Platform: "saleor",
		Language: "python",
		Python:   "3.12",
		DbType:   "PostgreSQL",
		Db:       "postgres:15",
		Redis:    "7.2.5",
	}
}
