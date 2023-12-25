package projects

import (
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/model/versions"
)

func PWA(config *configs.ConfigLines, defVersions versions.ToolsVersions, generalConf, projectConf map[string]string) {
	config.Set("NODEJS_ENABLED", "true")
	config.Set("NODEJS_VERSION", defVersions.NodeJs)
	config.Set("YARN_ENABLED", "true")
	config.Set("YARN_VERSION", defVersions.Yarn)
	config.Set("PWA_BACKEND_URL", defVersions.PwaBackendUrl)
	if _, ok := projectConf["PUBLIC_DIR"]; !ok {
		config.Set("PUBLIC_DIR", "")
	}
}
