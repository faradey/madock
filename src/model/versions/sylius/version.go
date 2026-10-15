package sylius

import (
	"github.com/faradey/madock/v4/src/model/versions"
)

// The matrix is data: src/model/versions/matrix/sylius.xml.

func init() {
	versions.RegisterProvider("sylius", GetVersions)
}

func GetVersions(ver string) versions.ToolsVersions {
	tv, _ := versions.FromMatrix("sylius", ver)
	return tv
}
