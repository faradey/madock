package versions

import (
	"github.com/faradey/madock/src/helper/ports"
)

// V310 migrates ports.conf from old format (project=number) to new format (project/service=port)
func V310() {
	registry := ports.GetRegistry()
	if registry.IsOldFormat() {
		registry.MigrateFromOldFormat()
	}
}
