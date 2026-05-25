package project

import (
	"github.com/faradey/madock/v3/src/helper/configs"
)

func init() {
	RegisterDockerConfGenerator("bigcommerce", MakeConfBigcommerce)
}

// MakeConfBigcommerce materialises only the Dockerfiles the selected
// preset uses. Mirror of MakeConfShopify — Node-only presets
// (catalyst, stencil, app-node) skip PHP/DB/Redis so we don't ship
// un-substituted templates that crash docker compose build.
func MakeConfBigcommerce(projectName string) {
	conf := configs.GetProjectConfig(projectName)
	phpEnabled := conf["php/enabled"] == "true"
	nodeEnabled := conf["nodejs/enabled"] == "true"
	dbEnabled := conf["db/type"] != "" && phpEnabled
	redisEnabled := conf["redis/enabled"] == "true"

	if phpEnabled {
		MakePhpDockerfile(projectName)
	}
	if nodeEnabled {
		MakeNodeJsDockerfile(projectName)
	}
	if dbEnabled {
		MakeDBDockerfile(projectName)
	}
	if redisEnabled {
		MakeRedisDockerfile(projectName)
	}
	MakeScriptsConf(projectName)
	MakeClaudeDockerfile(projectName)
}
