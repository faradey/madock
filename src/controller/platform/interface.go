package platform

import "fmt"

// Handler defines the interface for platform-specific operations
type Handler interface {
	// Start starts the containers for a project
	Start(projectName string, withChown bool, projectConf map[string]string)
	// Stop stops the containers for a project
	Stop(projectName string)
	// GetMainContainer returns the main container name for chown operations
	GetMainContainer() string
	// GetChownDirs returns directories to chown for this platform
	GetChownDirs(projectConf map[string]string) []string
	// SupportsCron returns whether this platform supports cron
	SupportsCron() bool
}

var handlers = map[string]Handler{}

// Register registers a platform handler
func Register(name string, handler Handler) {
	handlers[name] = handler
}

// Get returns the handler for the specified platform
func Get(platform string) (Handler, error) {
	h, ok := handlers[platform]
	if !ok {
		return nil, fmt.Errorf("unknown platform: %s", platform)
	}
	return h, nil
}

// GetOrDefault returns the handler for the specified platform,
// or the default handler if not found
func GetOrDefault(platform string) Handler {
	if h, ok := handlers[platform]; ok {
		return h
	}
	return &BaseHandler{}
}
