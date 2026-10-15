package bigcommerce

import (
	"github.com/faradey/madock/v4/src/model/versions"
)

// The stack is data: src/model/versions/matrix/bigcommerce.xml.

func init() {
	versions.RegisterProvider("bigcommerce", func(ver string) versions.ToolsVersions {
		tv, _ := versions.FromMatrix("bigcommerce", ver)
		return tv
	})
}

// GetVersions is the baseline stack, before any preset is chosen.
func GetVersions() versions.ToolsVersions {
	tv, _ := versions.FromMatrix("bigcommerce", "")
	return tv
}
