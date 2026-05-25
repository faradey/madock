package bigcommerce

import (
	"github.com/faradey/madock/v3/src/model/versions"
)

func init() {
	versions.RegisterProvider("bigcommerce", func(_ string) versions.ToolsVersions {
		return GetVersions()
	})
}

// GetVersions is the catch-all baseline used when no preset is
// selected. Real per-preset stacks come from GetBigcommercePresets.
func GetVersions() versions.ToolsVersions {
	return versions.ToolsVersions{
		Platform: "bigcommerce",
		Language: "nodejs",
		NodeJs:   "22.20.0",
		Yarn:     "1.22.22",
		Php:      "8.3",
		Db:       "mariadb:11.4",
		Composer: "2",
		Redis:    "7.4",
	}
}
