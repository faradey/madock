package shopware

import (
	"github.com/faradey/madock/src/model/versions"
	"strings"
)

func GetVersions(ver string) versions.ToolsVersions {
	swVersion := ""
	if ver != "" {
		swVersion = strings.TrimSpace(ver)
	}

	phpVer := GetPhpVersion(swVersion)
	return versions.ToolsVersions{
		Platform:        "shopware",
		Php:             phpVer,
		Db:              GetDBVersion(swVersion),
		SearchEngine:    GetSearchEngineVersion(swVersion),
		Elastic:         GetElasticVersion(swVersion),
		OpenSearch:      GetOpenSearchVersion(swVersion),
		Composer:        GetComposerVersion(swVersion),
		Redis:           GetRedisVersion(swVersion),
		RabbitMQ:        GetRabbitMQVersion(swVersion),
		Xdebug:          GetXdebugVersion(phpVer),
		PlatformVersion: swVersion,
		NodeJs:          "20.16.0",
		Yarn:            "3.6.4",
	}
}

func GetPhpVersion(mageVer string) string {
	if mageVer >= "6.5" {
		return "8.3"
	} else if mageVer >= "6.4" {
		return "8.1"
	}

	return ""
}

func GetDBVersion(mageVer string) string {
	if mageVer >= "6.5" {
		return "10.11"
	} else if mageVer >= "6.4" {
		return "10.5"
	}

	return ""
}

func GetElasticVersion(mageVer string) string {
	if mageVer >= "6.5" {
		return "8.11.14"
	} else if mageVer >= "6.4" {
		return "8.4.3"
	}

	return ""
}

func GetSearchEngineVersion(mageVer string) string {
	if mageVer >= "6.5" {
		return "OpenSearch"
	}

	return "Elasticsearch"
}

func GetOpenSearchVersion(mageVer string) string {
	if mageVer >= "6.5" {
		return "2.8.0"
	} else if mageVer >= "6.4" {
		return "2.5.0"
	}

	return ""
}

func GetComposerVersion(mageVer string) string {
	if mageVer >= "6.3" {
		return "2"
	}

	return ""
}

func GetRedisVersion(mageVer string) string {
	if mageVer >= "6.5" {
		return "7.2"
	}

	return ""
}

func GetRabbitMQVersion(mageVer string) string {
	if mageVer >= "6.3" {
		return "3.13"
	}

	return ""
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
