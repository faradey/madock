package woocommerce

import (
	"github.com/faradey/madock/v4/src/model/versions"
)

// The stack is data: src/model/versions/matrix/woocommerce.xml.

func init() {
	versions.RegisterProvider("woocommerce", GetVersions)
}

func GetVersions(ver string) versions.ToolsVersions {
	tv, _ := versions.FromMatrix("woocommerce", ver)
	return tv
}
