package projects

import (
	"strings"

	configs2 "github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/model/versions"
)

func init() {
	RegisterEnvWriter("bigcommerce", Bigcommerce)
}

// Bigcommerce presets map to different stack flavours:
//   - catalyst : Node-only (Next.js storefront)
//   - stencil  : Node-only (Stencil CLI for theme dev — proxies the
//                live store, no DB needed locally)
//   - api-php  : PHP + MariaDB + Redis (raw bigcommerce/api SDK)
//   - app-node : Node-only (Express + Next.js embedded app, OAuth
//                session storage via SQLite/JSON file by default)
//
// The preset is stored in `bigcommerce/preset` so it survives config
// rewrites and so install + setup controllers can branch on it.
func Bigcommerce(config *configs2.ConfigLines, defVersions versions.ToolsVersions, generalConf, projectConf map[string]string) {
	preset := defVersions.PlatformVersion
	if preset == "" {
		preset = configs2.GetOption("bigcommerce/preset", generalConf, projectConf)
	}
	if preset == "" {
		preset = "catalyst"
	}
	config.Set("bigcommerce/preset", preset)

	config.Set("timezone", configs2.GetOption("timezone", generalConf, projectConf))

	nodeOnly := preset == "catalyst" || preset == "stencil" || preset == "app-node"
	phpEnabled := !nodeOnly

	// PHP stack (api-php only).
	if phpEnabled {
		config.Set("php/enabled", "true")
		config.Set("php/version", defVersions.Php)
		config.Set("php/composer/version", defVersions.Composer)

		if _, ok := projectConf["public_dir"]; !ok {
			config.Set("public_dir", "public")
		}
		if _, ok := projectConf["composer_dir"]; !ok {
			config.Set("composer_dir", "")
		}

		config.Set("php/xdebug/version", versions.GetXdebugVersion(defVersions.Php))
		config.Set("php/xdebug/remote_host", "host.docker.internal")
		config.Set("php/xdebug/ide_key", configs2.GetOption("php/xdebug/ide_key", generalConf, projectConf))
		config.Set("php/xdebug/enabled", configs2.GetOption("php/xdebug/enabled", generalConf, projectConf))
		config.Set("php/ioncube/enabled", configs2.GetOption("php/ioncube/enabled", generalConf, projectConf))

		dbType, dbRepo := resolveDbTypeAndRepo(defVersions)
		config.Set("db/type", dbType)
		repoVersion := strings.Split(defVersions.Db, ":")
		if len(repoVersion) > 1 {
			config.Set("db/repository", repoVersion[0])
			config.Set("db/version", repoVersion[1])
		} else {
			if dbRepo != "" {
				config.Set("db/repository", dbRepo)
			}
			config.Set("db/version", defVersions.Db)
		}
		config.Set("db/root_password", configs2.GetOption("db/root_password", generalConf, projectConf))
		config.Set("db/user", configs2.GetOption("db/user", generalConf, projectConf))
		config.Set("db/password", configs2.GetOption("db/password", generalConf, projectConf))
		// Default database name for the PHP api-php preset.
		dbDatabase := configs2.GetOption("db/database", generalConf, projectConf)
		if dbDatabase == "" {
			dbDatabase = "bigcommerce"
		}
		config.Set("db/database", dbDatabase)

		// Redis on by default for PHP preset (session / cache).
		// Respect explicit project-level disable.
		redisEnabled := projectConf["redis/enabled"]
		if redisEnabled == "" {
			redisEnabled = "true"
		}
		config.Set("redis/enabled", redisEnabled)
		repoVersion = strings.Split(defVersions.Redis, ":")
		if len(repoVersion) > 1 {
			config.Set("redis/repository", repoVersion[0])
			config.Set("redis/version", repoVersion[1])
		} else {
			config.Set("redis/version", defVersions.Redis)
		}
	} else {
		config.Set("php/enabled", "false")
		config.Set("php/xdebug/enabled", "false")
		config.Set("redis/enabled", "false")
		config.Set("db/type", "")
	}

	// Node stack.
	nodeVer := defVersions.NodeJs
	if nodeVer == "" {
		nodeVer = "22.20.0"
	}
	if nodeOnly {
		config.Set("nodejs/enabled", "true")
		// Catalyst dev (Next.js) listens on 3000. Stencil CLI on
		// 3000. app-node on 3000. Match nginx upstream.
		config.Set("main_service_port", "3000")
		config.Set("nodejs/version", nodeVer)
		nodeMajorVersion := strings.Split(nodeVer, ".")
		if len(nodeMajorVersion) > 0 {
			config.Set("nodejs/major_version", nodeMajorVersion[0])
		}
	} else {
		// api-php — Node + Yarn still get baked into the PHP image
		// for asset pipelines / cli tooling users may want
		// alongside the SDK.
		config.Set("nodejs/enabled", "false")
		config.Set("php/nodejs/enabled", "true")
		config.Set("php/yarn/enabled", "true")
		config.Set("nodejs/version", nodeVer)
		nodeMajorVersion := strings.Split(nodeVer, ".")
		if len(nodeMajorVersion) > 0 {
			config.Set("nodejs/major_version", nodeMajorVersion[0])
		}
		// Reset stale Node-only setting if user switched away from
		// catalyst/stencil/app-node back to api-php.
		config.Set("main_service_port", "")
	}
	config.Set("nodejs/yarn/enabled", "true")
	if defVersions.Yarn != "" {
		config.Set("nodejs/yarn/version", defVersions.Yarn)
	}

	config.Set("grafana/auth/enabled", configs2.GetOption("grafana/auth/enabled", generalConf, projectConf))
	config.Set("grafana/auth/user", configs2.GetOption("grafana/auth/user", generalConf, projectConf))
	config.Set("grafana/auth/password", configs2.GetOption("grafana/auth/password", generalConf, projectConf))
}
