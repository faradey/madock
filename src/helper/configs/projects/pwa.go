package projects

import (
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/model/versions"
)

func PWA(config *configs.ConfigLines, defVersions versions.ToolsVersions, generalConf, projectConf map[string]string) {
	config.Set("nodejs/enabled", "true")
	config.Set("nodejs/version", defVersions.NodeJs)
	config.Set("nodejs/yarn/enabled", "true")
	config.Set("nodejs/yarn/version", defVersions.Yarn)
	config.Set("pwa/backend_url", defVersions.PwaBackendUrl)
	if _, ok := projectConf["public_dir"]; !ok {
		config.Set("public_dir", "")
	}
}
