package setup

import (
	"strings"

	"github.com/faradey/madock/v4/src/helper/configs"
)

// nameRefusal is why a directory cannot become a project under this name, or
// "" when it can.
//
// The Compose clash is checked after the config question on purpose: setting
// up an existing project again is not a clash with itself, and
// ComposeNameTakenBy already leaves the name itself out.
func nameRefusal(projectName string) string {
	if strings.Contains(projectName, ".") || strings.Contains(projectName, " ") {
		return "The project folder name cannot contain a period or space"
	}
	return configs.ComposeNameClash(projectName)
}
