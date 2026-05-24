package project

import (
	"github.com/faradey/madock/v3/src/helper/configs"
)

func init() {
	RegisterDockerConfGenerator("shopify", MakeConfShopify)
}

// MakeConfShopify materialises only the Dockerfiles the selected
// preset actually uses. Node-only presets (hydrogen, app-remix) skip
// PHP/DB/Redis so we don't ship un-substituted templates (e.g.
// `FROM mariadb:{{{db/version}}}`) that docker compose then tries to
// build and crash on.
func MakeConfShopify(projectName string) {
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
