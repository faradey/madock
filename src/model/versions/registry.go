package versions

// VersionProvider returns default ToolsVersions for a given platform version string.
type VersionProvider func(ver string) ToolsVersions

var providers = map[string]VersionProvider{}

// RegisterProvider registers a version matrix provider for a platform.
func RegisterProvider(platform string, fn VersionProvider) {
	providers[platform] = fn
}

// GetVersionsForPlatform returns default versions using the registered provider.
func GetVersionsForPlatform(platform, ver string) (ToolsVersions, bool) {
	if fn, ok := providers[platform]; ok {
		return fn(ver), true
	}
	return ToolsVersions{}, false
}
