package shopware

import (
	"github.com/faradey/madock/v4/src/model/versions"
)

// The matrix is data: src/model/versions/matrix/shopware.xml.

func init() {
	versions.RegisterProvider("shopware", GetVersions)
}

func GetVersions(ver string) versions.ToolsVersions {
	tv, _ := versions.FromMatrix("shopware", ver)
	return tv
}

func GetPhpVersion(v string) string { return versions.Answer("shopware", "php", v) }
