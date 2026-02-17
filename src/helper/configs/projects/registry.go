package projects

import (
	configs2 "github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/model/versions"
)

// EnvWriter writes platform-specific config entries.
type EnvWriter func(config *configs2.ConfigLines, defVersions versions.ToolsVersions, generalConf, projectConf map[string]string)

var envWriters = map[string]EnvWriter{}

// RegisterEnvWriter registers a config writer for a platform.
func RegisterEnvWriter(platform string, fn EnvWriter) {
	envWriters[platform] = fn
}

// GetEnvWriter returns the config writer for the given platform.
func GetEnvWriter(platform string) (EnvWriter, bool) {
	fn, ok := envWriters[platform]
	return fn, ok
}
