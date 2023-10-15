package magento2

import (
	"github.com/faradey/madock/src/versions"
	"os"
	"regexp"
	"strings"

	"github.com/faradey/madock/src/paths"
)

func GetVersions(ver string) versions.ToolsVersions {
	mageVersion := ""
	if ver == "" {
		_, mageVersion = getMagentoVersion()
	} else {
		mageVersion = strings.TrimSpace(ver)
	}

	phpVer := GetPhpVersion(mageVersion)
	return versions.ToolsVersions{
		Platform:     "magento2",
		Php:          phpVer,
		Db:           GetDBVersion(mageVersion),
		SearchEngine: GetSearchEngineVersion(mageVersion),
		Elastic:      GetElasticVersion(mageVersion),
		OpenSearch:   GetOpenSearchVersion(mageVersion),
		Composer:     GetComposerVersion(mageVersion),
		Redis:        GetRedisVersion(mageVersion),
		RabbitMQ:     GetRabbitMQVersion(mageVersion),
		Xdebug:       GetXdebugVersion(phpVer),
		Magento:      mageVersion,
	}
}

func getMagentoVersion() (edition, version string) {
	composerPath := paths.GetRunDirPath() + "/composer.json"
	txt, err := os.ReadFile(composerPath)
	if err == nil {
		re := regexp.MustCompile(`(?is)"magento/product-(community|enterprise)-edition".*?:.*?"[^0-9]*?([\.0-9]{5,}?(-p.*?|))"`)
		magentoVersion := re.FindAllStringSubmatch(string(txt), -1)
		if len(magentoVersion) > 0 && len(magentoVersion[0]) > 2 {
			return strings.TrimSpace(magentoVersion[0][1]), strings.TrimSpace(magentoVersion[0][2])
		}
	}

	return "", ""
}

func GetPhpVersion(mageVer string) string {
	if mageVer >= "2.4.4" {
		return "8.1"
	} else if mageVer >= "2.3.7" {
		return "7.4"
	} else if mageVer >= "2.3.3" {
		return "7.3"
	} else if mageVer >= "2.3.0" {
		return "7.2"
	} else if mageVer >= "2.2.0" {
		return "7.1"
	} else if mageVer >= "2.0.0" {
		return "7.0"
	}

	return ""
}

func GetDBVersion(mageVer string) string {
	if mageVer >= "2.4.6" {
		return "10.6"
	} else if mageVer >= "2.4.1" {
		return "10.4"
	} else if mageVer >= "2.3.7" {
		return "10.3"
	} else if mageVer >= "2.3.0" {
		return "10.2"
	} else if mageVer >= "2.3.3" {
		return "10.1"
	} else if mageVer >= "2.0.0" {
		return "10.0"
	}

	return ""
}

func GetElasticVersion(mageVer string) string {
	if mageVer >= "2.4.6" {
		return "8.4.3"
	} else if mageVer >= "2.4.5" {
		return "7.17.5"
	} else if mageVer >= "2.4.4" {
		return "7.16.3"
	} else if mageVer >= "2.4.3" {
		return "7.10.1"
	} else if mageVer >= "2.4.2" {
		return "7.9.3"
	} else if mageVer >= "2.4.1" {
		return "7.7.1"
	} else if mageVer >= "2.4.0" {
		return "7.6.2"
	} else if mageVer >= "2.3.7" {
		return "7.9.3"
	} else if mageVer >= "2.3.6" {
		return "7.7.1"
	} else if mageVer >= "2.3.5" {
		return "7.6.2"
	} else if mageVer >= "2.3.1" {
		return "6.8.13"
	} else if mageVer >= "2.0.0" {
		return "6.8.13"
	}

	return ""
}

func GetSearchEngineVersion(mageVer string) string {
	if mageVer >= "2.4.6" {
		return "OpenSearch"
	}

	return "Elasticsearch"
}

func GetOpenSearchVersion(mageVer string) string {
	if mageVer >= "2.4.6" {
		return "2.5.0"
	} else if mageVer >= "2.4.3-p2" {
		return "1.2.0"
	} else if mageVer == "2.3.7-p4" {
		return "1.2.0"
	} else if mageVer == "2.3.7-p3" {
		return "1.2.0"
	} else {
		return "NotCompatible"
	}

	return ""
}

func GetComposerVersion(mageVer string) string {
	if mageVer >= "2.4.2" {
		return "2"
	} else if mageVer >= "2.4.0" {
		return "1"
	} else if mageVer >= "2.3.7" {
		return "2"
	} else if mageVer >= "2.0.0" {
		return "1"
	}

	return ""
}

func GetRedisVersion(mageVer string) string {
	if mageVer >= "2.4.6" {
		return "7.0"
	} else if mageVer >= "2.4.4" {
		return "6.2"
	} else if mageVer >= "2.4.2" {
		return "6.0"
	} else if mageVer >= "2.4.0" {
		return "5.0"
	} else if mageVer >= "2.3.7" {
		return "6.0"
	} else if mageVer >= "2.0.0" {
		return "5.0"
	}

	return ""
}

func GetRabbitMQVersion(mageVer string) string {
	if mageVer >= "2.4.4" {
		return "3.9"
	} else if mageVer >= "2.3.4" {
		return "3.8"
	} else if mageVer >= "2.0.0" {
		return "3.7"
	}

	return ""
}

func GetXdebugVersion(phpVer string) string {
	if phpVer >= "8.1" {
		return "3.2.2"
	} else if phpVer >= "7.2" {
		return "3.1.6"
	}

	return "2.7.2"
}
