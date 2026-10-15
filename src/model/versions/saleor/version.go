package saleor

import (
	"github.com/faradey/madock/v4/src/model/versions"
)

// The stack is data: src/model/versions/matrix/saleor.xml.

func init() {
	versions.RegisterProvider("saleor", func(ver string) versions.ToolsVersions {
		tv, _ := versions.FromMatrix("saleor", ver)
		return tv
	})
}

// GetVersions is the baseline stack, before any preset is chosen.
func GetVersions() versions.ToolsVersions {
	tv, _ := versions.FromMatrix("saleor", "")
	return tv
}
