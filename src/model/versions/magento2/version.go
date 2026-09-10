package magento2

import (
	"os"
	"regexp"
	"strings"

	"github.com/faradey/madock/v4/src/model/versions"
)

// This package decides which tool versions a Magento release wants, and until
// 2026-09-10 it also went and found the release — reading `composer.json` from
// the run directory, which meant importing `helper/paths` from a model.
//
// That is a layer looking both ways: `helper/preset` and `helper/setup/tools`
// import this package, and this package imported a helper. Not a cycle Go would
// refuse, which is exactly why it survived — nothing complains, and the next
// helper this reaches for is the one that closes the loop for real.
//
// The reading now happens through a variable the caller may replace, defaulting
// to the working directory. The package still answers the same question; it no
// longer knows how the project is laid out on disk.

// ComposerJSON is where the Magento edition and version are read from when the
// caller does not name a version. Replaceable so that a caller with its own idea
// of the project's root — or a test — can say so.
var ComposerJSON = func() string {
	// MADOCK_RUN_DIR by hand, and the duplication is deliberate: reading it
	// through `helper/paths` is the import this change removes. The name is
	// pinned by a test that asks `paths.GetRunDirPath` for the same answer, so
	// the two cannot drift apart quietly — which is the only way a duplicated
	// constant hurts.
	if dir := strings.TrimSpace(os.Getenv("MADOCK_RUN_DIR")); dir != "" {
		return dir + "/composer.json"
	}

	dir, err := os.Getwd()
	if err != nil {
		return "composer.json"
	}

	return dir + "/composer.json"
}

func init() {
	versions.RegisterProvider("magento2", GetVersions)
}

func GetVersions(ver string) versions.ToolsVersions {
	mageVersion := ""
	if ver == "" {
		_, mageVersion = getMagentoVersion()
	} else {
		mageVersion = strings.TrimSpace(ver)
	}

	phpVer := GetPhpVersion(mageVersion)
	return versions.ToolsVersions{
		Platform:        "magento2",
		Language:        "php",
		Php:             phpVer,
		Db:              GetDBVersion(mageVersion),
		SearchEngine:    GetSearchEngineVersion(mageVersion),
		Elastic:         GetElasticVersion(mageVersion),
		OpenSearch:      GetOpenSearchVersion(mageVersion),
		Composer:        GetComposerVersion(mageVersion),
		Redis:           GetRedisVersion(mageVersion),
		Valkey:          GetValkeyVersion(mageVersion),
		RabbitMQ:        GetRabbitMQVersion(mageVersion),
		Xdebug:          versions.GetXdebugVersion(phpVer),
		PlatformVersion: mageVersion,
		NodeJs:          "18.15.0",
		Yarn:            "3.6.4",
	}
}

func getMagentoVersion() (edition, version string) {
	composerPath := ComposerJSON()
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
	if mageVer >= "2.4.9" {
		return "8.5"
	} else if mageVer >= "2.4.8" {
		return "8.4"
	} else if mageVer >= "2.4.7" {
		return "8.3"
	} else if mageVer >= "2.4.4" {
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
	if mageVer >= "2.4.9" {
		return "11.8"
	} else if mageVer >= "2.4.8" {
		return "11.4"
	} else if mageVer >= "2.4.7" {
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
	if mageVer >= "2.4.8" {
		return "8.17.6"
	} else if mageVer >= "2.4.7" {
		return "8.11.14"
	} else if mageVer >= "2.4.6" {
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
	if mageVer >= "2.4.9" {
		return "3.0.0"
	} else if mageVer >= "2.4.8" {
		return "2.19.0"
	} else if mageVer >= "2.4.7" {
		return "2.12.0"
	} else if mageVer >= "2.4.6" {
		return "2.5.0"
	} else if mageVer >= "2.4.3-p2" {
		return "1.2.0"
	} else if mageVer == "2.3.7-p4" {
		return "1.2.0"
	} else if mageVer == "2.3.7-p3" {
		return "1.2.0"
	}

	return "NotCompatible"
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
	if mageVer >= "2.4.8" {
		return "8.0"
	} else if mageVer >= "2.4.7" {
		return "7.2"
	} else if mageVer >= "2.4.6" {
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

func GetValkeyVersion(mageVer string) string {
	if mageVer >= "2.4.9" {
		return "9.0.0"
	}
	return "8.1.3"
}

func GetRabbitMQVersion(mageVer string) string {
	if mageVer >= "2.4.9" {
		return "4.2"
	} else if mageVer >= "2.4.7-p5" {
		return "4.1"
	} else if mageVer >= "2.4.7" {
		return "3.13"
	} else if mageVer >= "2.4.6" {
		return "3.11"
	} else if mageVer >= "2.4.4" {
		return "3.9"
	} else if mageVer >= "2.3.4" {
		return "3.8"
	} else if mageVer >= "2.0.0" {
		return "3.7"
	}

	return ""
}
