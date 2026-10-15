package spree

import (
	"github.com/faradey/madock/v4/src/model/versions"
)

// The stack is data: src/model/versions/matrix/spree.xml.

func init() {
	versions.RegisterProvider("spree", func(ver string) versions.ToolsVersions {
		tv, _ := versions.FromMatrix("spree", ver)
		return tv
	})
}

// GetVersions is the baseline stack, before any preset is chosen.
func GetVersions() versions.ToolsVersions {
	tv, _ := versions.FromMatrix("spree", "")
	return tv
}
