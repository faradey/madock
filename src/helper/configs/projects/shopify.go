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
	nodeEnabled := nodeOnly || preset == "laravel-shopify"

	// PHP stack (api-php, laravel-shopify, legacy).
	if phpEnabled {
		config.Set("php/enabled", "true")
		config.Set("php/version", defVersions.Php)
		config.Set("php/composer/version", defVersions.Composer)

		// Hydrogen / Remix presets serve from the project root; PHP
		// presets use Symfony/Laravel-style `public/` (api-php legacy
		// kept `web/public`). Laravel-shopify uses `public/` by
		// convention.
		if _, ok := projectConf["public_dir"]; !ok {
			if preset == "laravel-shopify" {
				config.Set("public_dir", "public")
			} else {
				config.Set("public_dir", "web/public")
			}
		}
		if _, ok := projectConf["composer_dir"]; !ok {
			if preset == "laravel-shopify" {
				config.Set("composer_dir", "")
			} else {
				config.Set("composer_dir", "web")
			}
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

		config.Set("redis/enabled", "true")
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
	}

	// Node stack (hydrogen, app-remix, laravel-shopify for asset
	// pipeline).
	if nodeEnabled {
		// For Hydrogen / app-remix the nodejs container IS the main
		// service. For laravel-shopify Node is just the asset
		// pipeline embedded into the PHP image.
		if nodeOnly {
			config.Set("nodejs/enabled", "true")
			// Hydrogen dev server listens on 3000, app-remix uses 3000
			// (via shopify CLI tunnel). Match nginx upstream.
			config.Set("main_service_port", "3000")
		} else {
			config.Set("php/nodejs/enabled", "true")
			config.Set("php/yarn/enabled", "true")
		}
		config.Set("nodejs/version", defVersions.NodeJs)
		nodeMajorVersion := strings.Split(defVersions.NodeJs, ".")
		if len(nodeMajorVersion) > 0 {
			config.Set("nodejs/major_version", nodeMajorVersion[0])
		}
		config.Set("nodejs/yarn/enabled", "true")
		if defVersions.Yarn != "" {
			config.Set("nodejs/yarn/version", defVersions.Yarn)
		}
	} else {
		config.Set("nodejs/enabled", "false")
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
