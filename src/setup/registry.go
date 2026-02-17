package setup

import (
	"sort"

	"github.com/faradey/madock/src/helper/cli/arg_struct"
)

// SetupContext provides all data a platform setup handler needs.
type SetupContext struct {
	ProjectName     string
	ProjectConf     map[string]string
	ContinueSetup  bool
	Args            *arg_struct.ControllerGeneralSetup
	DetectedVersion string
	Language        string
}

// SetupHandler is implemented by each platform to handle the "setup" command.
type SetupHandler interface {
	Execute(ctx *SetupContext)
}

// PlatformInfo describes a platform available for setup.
type PlatformInfo struct {
	Name        string // internal name, e.g. "magento2"
	DisplayName string // human-readable name, e.g. "Magento 2"
	Language    string // default language; empty means prompt user
	Order       int    // display order in the interactive wizard
}

var handlers = map[string]SetupHandler{}
var platforms []PlatformInfo

// Register adds a platform setup handler to the registry.
func Register(info PlatformInfo, handler SetupHandler) {
	handlers[info.Name] = handler
	platforms = append(platforms, info)
	sort.Slice(platforms, func(i, j int) bool {
		return platforms[i].Order < platforms[j].Order
	})
}

// Get returns the setup handler for the given platform name.
func Get(name string) (SetupHandler, bool) {
	h, ok := handlers[name]
	return h, ok
}

// GetPlatformInfo returns platform metadata by name.
func GetPlatformInfo(name string) (PlatformInfo, bool) {
	for _, p := range platforms {
		if p.Name == name {
			return p, true
		}
	}
	return PlatformInfo{}, false
}

// Platforms returns all registered platforms in order.
func Platforms() []PlatformInfo {
	return platforms
}

// PlatformNames returns platform names in display order.
func PlatformNames() []string {
	names := make([]string, len(platforms))
	for i, p := range platforms {
		names[i] = p.Name
	}
	return names
}
