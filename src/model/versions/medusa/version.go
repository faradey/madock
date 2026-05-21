package medusa

import (
	"github.com/faradey/madock/v3/src/model/versions"
)

func init() {
	versions.RegisterProvider("medusa", func(_ string) versions.ToolsVersions {
		return GetVersions()
	})
}

func GetVersions() versions.ToolsVersions {
	return versions.ToolsVersions{
		Platform: "medusa",
		Language: "nodejs",
		DbType:   "PostgreSQL",
		Db:       "postgres:16.4",
		Redis:    "7.2.1",
		NodeJs:   "20.18.0",
		Yarn:     "4.5.0",
	}
}
