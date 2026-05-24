package sylius

import (
	"strings"

	"github.com/faradey/madock/v3/src/model/versions"
)

func init() {
	versions.RegisterProvider("sylius", GetVersions)
}

func GetVersions(ver string) versions.ToolsVersions {
	syVersion := strings.TrimSpace(ver)
	phpVer := GetPhpVersion(syVersion)

	return versions.ToolsVersions{
		Platform:        "sylius",
		Language:        "php",
		Php:             phpVer,
		Db:              GetDBVersion(syVersion),
		Composer:        "2",
		Redis:           GetRedisVersion(syVersion),
		Xdebug:          GetXdebugVersion(phpVer),
		PlatformVersion: syVersion,
		NodeJs:          "22.20.0",
		Yarn:            "1.22.22",
	}
}

func GetPhpVersion(v string) string {
	if v >= "2.0" {
		return "8.3"
	}
	if v >= "1.13" {
		return "8.2"
	}
	if v >= "1.12" {
		return "8.1"
	}
	return "8.3"
}

func GetDBVersion(v string) string {
	if v >= "2.0" {
		return "mariadb:11.4"
	}
	if v >= "1.13" {
		return "mariadb:10.11"
	}
	return "mariadb:10.6"
}

func GetRedisVersion(v string) string {
	if v >= "2.0" {
		return "7.4"
	}
	return "7.2"
}

func GetXdebugVersion(phpVer string) string {
	if phpVer >= "8.4" {
		return "3.4.4"
	} else if phpVer >= "8.3" {
		return "3.3.1"
	} else if phpVer >= "8.1" {
		return "3.2.2"
	}
	return "3.1.6"
}
