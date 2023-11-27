package pwa

import (
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/model/versions"
)

func GetVersions() versions.ToolsVersions {
	projectConf := configs.GetGeneralConfig()
	return versions.ToolsVersions{
		Platform: "pwa",
		NodeJs:   projectConf["NODE_VERSION"],
		Yarn:     projectConf["YARN_VERSION"],
	}
}
