package versions

import (
	"github.com/faradey/madock/src/paths"
	"io/ioutil"
	"regexp"
)

type ToolsVersions struct {
	Php, Db, Elastic, Composer, Redis, RabbitMQ, Xdebug string
}

func GetVersions() ToolsVersions {
	_, mageVersion := getMagentoVersion()
	return ToolsVersions{
		Php:      GetPhpVersion(mageVersion),
		Db:       GetDBVersion(mageVersion),
		Elastic:  GetElasticVersion(mageVersion),
		Composer: GetComposerVersion(mageVersion),
		Redis:    GetRedisVersion(mageVersion),
		RabbitMQ: GetRabbitMQVersion(mageVersion),
		Xdebug:   GetXdebugVersion(mageVersion),
	}
}

func getMagentoVersion() (edition, version string) {
	composerPath := paths.GetRunDirPath() + "/composer.json"
	txt, err := ioutil.ReadFile(composerPath)
	if err == nil {
		re := regexp.MustCompile(`(?is)"magento/product-(community|enterprise)-edition".*?:.*?"([\.0-9]{5,}?)-*.*?"`)
		magentoVersion := re.FindAllStringSubmatch(string(txt), -1)
		if len(magentoVersion) > 0 && len(magentoVersion[0]) > 2 {
			return magentoVersion[0][1], magentoVersion[0][2]
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
	if mageVer >= "2.4.1" {
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
	if mageVer >= "2.4.4" {
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
	if mageVer >= "2.4.2" {
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
	if mageVer >= "2.3.4" {
		return "3.8"
	} else if mageVer >= "2.0.0" {
		return "3.7"
	}

	return ""
}

func GetXdebugVersion(mageVer string) string {
	if mageVer < "2.4" {
		return "2.9.8"
	}

	return "3"
}
