package magento2

import (
	"os"
	"regexp"
	"strings"

	"github.com/faradey/madock/v4/src/model/versions"
)

// The matrix itself — which tool versions a Magento release wants — is data in
// src/model/versions/matrix/magento2.xml. What stays here is finding the
// release when the caller does not name one.
//
// Until 2026-09-10 this package also went and found the release by importing
// `helper/paths` from a model, a layer looking both ways. The reading happens
// through a variable the caller may replace, defaulting to the working
// directory.

// ComposerJSON is where the Magento edition and version are read from when the
// caller does not name a version. Replaceable so that a caller with its own idea
// of the project's root — or a test — can say so.
var ComposerJSON = func() string {
	// MADOCK_RUN_DIR by hand, and the duplication is deliberate: reading it
	// through `helper/paths` is the import this package must not have. The name
	// is pinned by a test that asks `paths.GetRunDirPath` for the same answer,
	// so the two cannot drift apart quietly.
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
	mageVersion := strings.TrimSpace(ver)
	if mageVersion == "" {
		_, mageVersion = getMagentoVersion()
	}

	tv, _ := versions.FromMatrix("magento2", mageVersion)
	return tv
}

func getMagentoVersion() (edition, version string) {
	txt, err := os.ReadFile(ComposerJSON())
	if err == nil {
		re := regexp.MustCompile(`(?is)"magento/product-(community|enterprise)-edition".*?:.*?"[^0-9]*?([\.0-9]{5,}?(-p.*?|))"`)
		magentoVersion := re.FindAllStringSubmatch(string(txt), -1)
		if len(magentoVersion) > 0 && len(magentoVersion[0]) > 2 {
			return strings.TrimSpace(magentoVersion[0][1]), strings.TrimSpace(magentoVersion[0][2])
		}
	}

	return "", ""
}

// One tool at a time, for the callers and tests that ask for one.

func GetPhpVersion(v string) string { return versions.Answer("magento2", "php", v) }
func GetDBVersion(v string) string  { return versions.Answer("magento2", "db", v) }
func GetElasticVersion(v string) string {
	return versions.Answer("magento2", "elasticsearch", v)
}
func GetSearchEngineVersion(v string) string {
	return versions.Answer("magento2", "searchengine", v)
}
func GetOpenSearchVersion(v string) string {
	return versions.Answer("magento2", "opensearch", v)
}
func GetComposerVersion(v string) string { return versions.Answer("magento2", "composer", v) }
func GetRedisVersion(v string) string    { return versions.Answer("magento2", "redis", v) }
func GetValkeyVersion(v string) string   { return versions.Answer("magento2", "valkey", v) }
func GetRabbitMQVersion(v string) string { return versions.Answer("magento2", "rabbitmq", v) }
