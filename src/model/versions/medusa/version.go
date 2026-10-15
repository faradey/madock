package medusa

import (
	"github.com/faradey/madock/v4/src/model/versions"
)

// The stack is data: src/model/versions/matrix/medusa.xml.

func init() {
	versions.RegisterProvider("medusa", func(ver string) versions.ToolsVersions {
		tv, _ := versions.FromMatrix("medusa", ver)
		return tv
	})
}

// GetVersions is the baseline stack, before any preset is chosen.
func GetVersions() versions.ToolsVersions {
	tv, _ := versions.FromMatrix("medusa", "")
	return tv
}
