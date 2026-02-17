package shopify

import (
	"github.com/faradey/madock/v3/src/model/versions"
)

func init() {
	versions.RegisterProvider("shopify", func(_ string) versions.ToolsVersions {
		return GetVersions()
	})
}

func GetVersions() versions.ToolsVersions {
	return versions.ToolsVersions{
		Platform: "shopify",
		Language: "php",
		Php:      "8.2",
		Db:       "11.1.2",
		Composer: "2",
		Redis:    "7.2.1",
		Valkey:   "8.1.3",
		RabbitMQ: "3.9.29",
		Xdebug:   "3.2.2",
		NodeJs:   "18.15.0",
		Yarn:     "3.6.4",
	}
}
