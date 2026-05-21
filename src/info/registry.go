// Package info provides a registry for platform-specific "info" handlers.
// Platform packages register an InfoHandler in init(); the general/info
// controller resolves the handler by project platform and falls back to a
// generic printer when no handler is registered.
package info

// InfoContext provides all data a platform info handler needs.
type InfoContext struct {
	ProjectName string
	ProjectPath string
	ProjectConf map[string]string
	Service     string
}

// InfoHandler is implemented by each platform to handle the "info" command.
type InfoHandler interface {
	Print(ctx *InfoContext) error
}

var handlers = map[string]InfoHandler{}

// Register adds a platform info handler to the registry.
func Register(platform string, handler InfoHandler) {
	handlers[platform] = handler
}

// Get returns the info handler for the given platform name.
func Get(platform string) (InfoHandler, bool) {
	h, ok := handlers[platform]
	return h, ok
}
