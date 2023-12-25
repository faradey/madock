package projects

import (
	configs2 "github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/model/versions"
)

func Custom(config *configs2.ConfigLines, defVersions versions.ToolsVersions, generalConf, projectConf map[string]string) {
	if _, ok := projectConf["public_dir"]; !ok {
		config.Set("public_dir", "web/public")
	}
}
