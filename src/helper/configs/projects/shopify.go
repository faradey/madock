package projects

import (
	"strings"

	configs2 "github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/model/versions"
)

func init() {
	RegisterEnvWriter("shopify", Shopify)
}

// Shopify presets map to different stack flavours:
//   - hydrogen         : Node-only (Hydrogen storefront)
//   - app-remix        : Node-only (Shopify App Remix template; Prisma
//                        uses SQLite by default, no DB container needed)
//   - api-php          : PHP + MariaDB + Redis (raw shopify-api SDK)
//   - laravel-shopify  : PHP + Node + MariaDB + Redis (full Laravel app)
//   - <empty / legacy> : PHP + Node + MariaDB + Redis (backwards-
//                        compatible default)
//
// The preset is stored in `shopify/preset` so it survives config
// rewrites and so install + setup controllers can branch on it.
func Shopify(config *configs2.ConfigLines, defVersions versions.ToolsVersions, generalConf, projectConf map[string]string) {
	preset := defVersions.PlatformVersion
	if preset == "" {
		preset = configs2.GetOption("shopify/preset", generalConf, projectConf)
	}
	if preset == "" {
		preset = "api-php"
	}
	config.Set("shopify/preset", preset)

	config.Set("timezone", configs2.GetOption("timezone", generalConf, projectConf))

	nodeOnly := preset == "hydrogen" || preset == "app-remix"
	phpEnabled := !nodeOnly

	// PHP stack (api-php, laravel-shopify, legacy).
	if phpEnabled {
		config.Set("php/enabled", "true")
		config.Set("php/version", defVersions.Php)
		config.Set("php/composer/version", defVersions.Composer)

		// Web roots:
		//   laravel-shopify : Laravel default `public/`, composer
		//                     at root
		//   api-php         : composer init scaffolds at the root
		//                     (no `web/` subdir); the SDK is a
		//                     library, not a framework — `public/`
		//                     stays empty unless the user writes a
		//                     front controller
		// Legacy projects (previous madock Shopify-PHP template
		// using `web/public/` + composer at `web/`) keep their old
		// values because the `!ok` check preserves whatever's
		// already in projectConf.
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
		config.Set("db/database", configs2.GetOption("db/database", generalConf, projectConf))

		// Redis on by default for PHP presets — Laravel +
		// shopify-api SDK both ship Redis-backed session/cache
		// helpers. Project-level `<redis><enabled>false</enabled>`
		// can still disable explicitly via setup wizard, but the
		// global default (off) is ignored here.
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
		// Hydrogen + app-remix don't need a database — Hydrogen calls
		// the Storefront API directly, app-remix uses Prisma+SQLite by
		// default. Skip db/redis containers entirely.
		config.Set("redis/enabled", "false")
		// Clear the PHP front-controller pointers in case the user
		// previously had a PHP-stack preset configured. nginx for
		// Node presets ignores these, but db:export and similar
		// helpers may still consult them.
		config.Set("db/type", "")
	}

	// Node stack. Two flavours:
	//   - nodeOnly (hydrogen, app-remix) - dedicated nodejs container
	//                                       is the main service
	//   - PHP-stack presets - Node + Yarn baked into the PHP image
	//                          so asset pipelines (Webpack Encore,
	//                          Mix, Vite, the legacy Shopify
	//                          PHP-template's web/frontend) keep
	//                          working. Includes api-php for
	//                          backwards compatibility — pre-branch
	//                          shopify projects had node + yarn in
	//                          the PHP image, removing them would
	//                          break existing tooling
	nodeVer := defVersions.NodeJs
	if nodeVer == "" {
		nodeVer = "22.20.0"
	}
	if nodeOnly {
		config.Set("nodejs/enabled", "true")
		// Hydrogen dev server listens on 3000, app-remix uses 3000
		// (via shopify CLI tunnel). Match nginx upstream.
		config.Set("main_service_port", "3000")
		config.Set("nodejs/version", nodeVer)
		nodeMajorVersion := strings.Split(nodeVer, ".")
		if len(nodeMajorVersion) > 0 {
			config.Set("nodejs/major_version", nodeMajorVersion[0])
		}
	} else {
		// PHP-stack preset (api-php / laravel-shopify / legacy)
		config.Set("nodejs/enabled", "false")
		config.Set("php/nodejs/enabled", "true")
		config.Set("php/yarn/enabled", "true")
		config.Set("nodejs/version", nodeVer)
		nodeMajorVersion := strings.Split(nodeVer, ".")
		if len(nodeMajorVersion) > 0 {
			config.Set("nodejs/major_version", nodeMajorVersion[0])
		}
		// Reset stale Node-only setting if user switched away from
		// hydrogen/app-remix back to a PHP preset.
		config.Set("main_service_port", "")
	}
	config.Set("nodejs/yarn/enabled", "true")
	if defVersions.Yarn != "" {
		config.Set("nodejs/yarn/version", defVersions.Yarn)
	}

	// RabbitMQ + Grafana stay opt-in across all presets.
	config.Set("rabbitmq/enabled", configs2.GetOption("rabbitmq/enabled", generalConf, projectConf))
	repoVersion := strings.Split(defVersions.RabbitMQ, ":")
	if len(repoVersion) > 1 {
		config.Set("rabbitmq/repository", repoVersion[0])
		config.Set("rabbitmq/version", repoVersion[1])
	} else if defVersions.RabbitMQ != "" {
		config.Set("rabbitmq/version", defVersions.RabbitMQ)
	}
	config.Set("rabbitmq/user", configs2.GetOption("rabbitmq/user", generalConf, projectConf))
	config.Set("rabbitmq/password", configs2.GetOption("rabbitmq/password", generalConf, projectConf))

	config.Set("grafana/auth/enabled", configs2.GetOption("grafana/auth/enabled", generalConf, projectConf))
	config.Set("grafana/auth/user", configs2.GetOption("grafana/auth/user", generalConf, projectConf))
	config.Set("grafana/auth/password", configs2.GetOption("grafana/auth/password", generalConf, projectConf))
}
