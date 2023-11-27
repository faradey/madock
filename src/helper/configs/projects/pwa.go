package projects

import (
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/model/versions"
)

func PWA(config *configs.ConfigLines, defVersions versions.ToolsVersions, generalConf, projectConf map[string]string) {
	config.AddOrSetLine("NODEJS_ENABLED", "true")
	config.AddOrSetLine("NODEJS_VERSION", defVersions.NodeJs)
	config.AddOrSetLine("YARN_ENABLED", "true")
	config.AddOrSetLine("YARN_VERSION", defVersions.Yarn)
	config.AddOrSetLine("PWA_BACKEND_URL", defVersions.PwaBackendUrl)
}
