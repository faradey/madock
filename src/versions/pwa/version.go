package pwa

import (
	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/versions"
)

func GetVersions() versions.ToolsVersions {
	projectConf := configs.GetGeneralConfig()
	return versions.ToolsVersions{
		Platform: "pwa",
		NodeJs:   projectConf["NODE_VERSION"],
		Yarn:     projectConf["YARN_VERSION"],
	}
}
