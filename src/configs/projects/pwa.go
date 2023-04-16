package projects

import (
	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/versions"
)

func PWA(config *configs.ConfigLines, defVersions versions.ToolsVersions, generalConf, projectConfig map[string]string) {
	config.AddOrSetLine("NODEJS_ENABLED", "true")
	config.AddOrSetLine("NODE_VERSION", defVersions.NodeJs)
	config.AddOrSetLine("YARN_ENABLED", "true")
	config.AddOrSetLine("YARN_VERSION", defVersions.Yarn)
}
