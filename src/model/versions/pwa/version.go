package pwa

import (
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/model/versions"
)

func GetVersions() versions.ToolsVersions {
	projectConf := configs.GetGeneralConfig()
	return versions.ToolsVersions{
		Platform: "pwa",
		Language: "nodejs",
		NodeJs:   projectConf["nodejs/version"],
		Yarn:     projectConf["nodejs/yarn/version"],
	}
}
