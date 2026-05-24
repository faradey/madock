package spree

import (
	"github.com/faradey/madock/v3/src/model/versions"
)

func init() {
	versions.RegisterProvider("spree", func(_ string) versions.ToolsVersions {
		return GetVersions()
	})
}

func GetVersions() versions.ToolsVersions {
	return versions.ToolsVersions{
		Platform: "spree",
		Language: "ruby",
		Ruby:     "4.0.5",
		DbType:   "PostgreSQL",
		Db:       "postgres:16.4",
		Redis:    "7.2.5",
	}
}
