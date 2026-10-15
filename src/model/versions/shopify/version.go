package shopify

import (
	"github.com/faradey/madock/v4/src/model/versions"
)

// The stack is data: src/model/versions/matrix/shopify.xml.

func init() {
	versions.RegisterProvider("shopify", func(ver string) versions.ToolsVersions {
		tv, _ := versions.FromMatrix("shopify", ver)
		return tv
	})
}

// GetVersions is the baseline stack, before any preset is chosen.
func GetVersions() versions.ToolsVersions {
	tv, _ := versions.FromMatrix("shopify", "")
	return tv
}
