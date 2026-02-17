package custom

import (
	"github.com/faradey/madock/v3/src/model/versions"
	"github.com/faradey/madock/v3/src/model/versions/languages"
)

func init() {
	versions.RegisterProvider("custom", func(_ string) versions.ToolsVersions {
		return GetVersions()
	})
}

func GetVersions() versions.ToolsVersions {
	phpVer := GetPhpVersion()
	langDefaults := languages.GetAllDefaults()
	return versions.ToolsVersions{
		Platform:     "custom",
		Language:     "php",
		Php:          phpVer,
		Db:           GetDBVersion(),
		SearchEngine: GetSearchEngineVersion(),
		Elastic:      GetElasticVersion(),
		OpenSearch:   GetOpenSearchVersion(),
		Composer:     GetComposerVersion(),
		Redis:        GetRedisVersion(),
		Valkey:       GetValkeyVersion(),
		RabbitMQ:     GetRabbitMQVersion(),
		Xdebug:       GetXdebugVersion(phpVer),
		Python:       langDefaults["python"],
		Golang:       langDefaults["golang"],
		Ruby:         langDefaults["ruby"],
	}
}

func GetPhpVersion() string {
	return "8.2"
}

func GetDBVersion() string {
	return "10.6"
}

func GetElasticVersion() string {
	return "8.4.3"
}

func GetSearchEngineVersion() string {
	return "Elasticsearch"
}

func GetOpenSearchVersion() string {
	return "2.5.0"
}

func GetComposerVersion() string {
	return "2"
}

func GetRedisVersion() string {
	return "7.0"
}

func GetValkeyVersion() string {
	return "8.1.3"
}

func GetRabbitMQVersion() string {
	return "3.9"
}

func GetXdebugVersion(phpVer string) string {
	if phpVer >= "8.4" {
		return "3.4.4"
	} else if phpVer >= "8.3" {
		return "3.3.1"
	} else if phpVer >= "8.1" {
		return "3.2.2"
	} else if phpVer >= "7.2" {
		return "3.1.6"
	}

	return "2.7.2"
}
