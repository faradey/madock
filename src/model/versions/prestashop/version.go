package prestashop

import (
	"strings"

	"github.com/faradey/madock/src/model/versions"
)

func GetVersions(ver string) versions.ToolsVersions {
	platformVer := ""
	if ver != "" {
		platformVer = strings.TrimSpace(ver)
	}

	phpVer := GetPhpVersion()
	return versions.ToolsVersions{
		Platform:        "prestashop",
		Language:        "php",
		Php:             phpVer,
		Db:              GetDBVersion(),
		SearchEngine:    GetSearchEngineVersion(),
		Elastic:         GetElasticVersion(),
		OpenSearch:      GetOpenSearchVersion(),
		Composer:        GetComposerVersion(),
		Redis:           GetRedisVersion(),
		Valkey:          GetValkeyVersion(),
		RabbitMQ:        GetRabbitMQVersion(),
		Xdebug:          versions.GetXdebugVersion(phpVer),
		PlatformVersion: platformVer,
		NodeJs:          "18.15.0",
		Yarn:            "3.6.4",
	}
}

func GetPhpVersion() string {
	return "8.1"
}

func GetDBVersion() string {
	return "10.6"

}

func GetElasticVersion() string {
	return "8.11.14"
}

func GetSearchEngineVersion() string {
	return "Elasticsearch"
}

func GetOpenSearchVersion() string {
	return "2.12.0"
}

func GetComposerVersion() string {
	return "2"
}

func GetRedisVersion() string {
	return "7.2"
}

func GetValkeyVersion() string {
	return "8.1.3"
}

func GetRabbitMQVersion() string {
	return "3.13"
}
