package custom

import (
	"github.com/faradey/madock/v4/src/model/versions"
	"github.com/faradey/madock/v4/src/model/versions/languages"
)

// The stack is data: src/model/versions/matrix/custom.xml. Python, Go and Ruby
// come from the languages package, which knows them for every language madock
// supports.

func init() {
	versions.RegisterProvider("custom", func(_ string) versions.ToolsVersions {
		return GetVersions()
	})
}

func GetVersions() versions.ToolsVersions {
	tv, _ := versions.FromMatrix("custom", "")
	langDefaults := languages.GetAllDefaults()
	tv.Python = langDefaults["python"]
	tv.Golang = langDefaults["golang"]
	tv.Ruby = langDefaults["ruby"]
	return tv
}
