package prestashop

import (
	"github.com/faradey/madock/v4/src/model/versions"
)

// The stack is data: src/model/versions/matrix/prestashop.xml.

func init() {
	versions.RegisterProvider("prestashop", GetVersions)
}

func GetVersions(ver string) versions.ToolsVersions {
	tv, _ := versions.FromMatrix("prestashop", ver)
	return tv
}
