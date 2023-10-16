package shopify

import (
	"github.com/faradey/madock/src/versions"
)

func GetVersions() versions.ToolsVersions {
	return versions.ToolsVersions{
		Platform: "shopify",
		Php:      "8.2",
		Db:       "11.1.2",
		Composer: "2",
		Redis:    "7.2.1",
		RabbitMQ: "3.9.29",
		Xdebug:   "3.2.2",
		NodeJs:   "18.15.0",
		Yarn:     "3.6.4",
	}
}
