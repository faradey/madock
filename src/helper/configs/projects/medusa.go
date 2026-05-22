package projects

import (
	"strings"

	configs2 "github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/model/versions"
)

func init() {
	RegisterEnvWriter("medusa", Medusa)
}

func Medusa(config *configs2.ConfigLines, defVersions versions.ToolsVersions, generalConf, projectConf map[string]string) {
	if _, ok := projectConf["public_dir"]; !ok {
		config.Set("public_dir", "")
	}
	if _, ok := projectConf["composer_dir"]; !ok {
		config.Set("composer_dir", "")
	}

	config.Set("php/enabled", "false")
	config.Set("php/xdebug/enabled", "false")
	config.Set("python/enabled", "false")
	config.Set("golang/enabled", "false")
	config.Set("ruby/enabled", "false")

	// Medusa backend listens on port 9000. Make sure the nginx proxy.conf
	// template uses it as the upstream port instead of the default 3000.
	config.Set("main_service_port", "9000")

	config.Set("nodejs/enabled", "true")
	config.Set("nodejs/version", defVersions.NodeJs)
	nodeMajorVersion := strings.Split(defVersions.NodeJs, ".")
	if len(nodeMajorVersion) > 0 {
		config.Set("nodejs/major_version", nodeMajorVersion[0])
	}
	config.Set("nodejs/yarn/enabled", "true")
	config.Set("nodejs/yarn/version", defVersions.Yarn)
	config.Set("timezone", configs2.GetOption("timezone", generalConf, projectConf))

	// Medusa requires PostgreSQL; force the type but allow user-overridden version.
	if defVersions.DbType == "" {
		defVersions.DbType = "PostgreSQL"
	}
	dbType, dbRepo := resolveDbTypeAndRepo(defVersions)
	config.Set("db/type", dbType)

	dbVersion := defVersions.Db
	if dbVersion == "" {
		dbVersion = configs2.GetOption("db/version", generalConf, projectConf)
	}
	repoVersion := strings.Split(dbVersion, ":")
	if len(repoVersion) > 1 {
		config.Set("db/repository", repoVersion[0])
		config.Set("db/version", repoVersion[1])
	} else {
		if dbRepo != "" {
			config.Set("db/repository", dbRepo)
		}
		config.Set("db/version", dbVersion)
	}
	config.Set("db/root_password", configs2.GetOption("db/root_password", generalConf, projectConf))
	config.Set("db/user", configs2.GetOption("db/user", generalConf, projectConf))
	config.Set("db/password", configs2.GetOption("db/password", generalConf, projectConf))
	config.Set("db/database", configs2.GetOption("db/database", generalConf, projectConf))

	config.Set("search/elasticsearch/enabled", "false")
	config.Set("search/opensearch/enabled", "false")

	// Medusa uses Redis for events/workflow state.
	config.Set("redis/enabled", "true")
	redisVersion := defVersions.Redis
	if redisVersion == "" {
		redisVersion = configs2.GetOption("redis/version", generalConf, projectConf)
	}
	repoVersion = strings.Split(redisVersion, ":")
	if len(repoVersion) > 1 {
		config.Set("redis/repository", repoVersion[0])
		config.Set("redis/version", repoVersion[1])
	} else {
		config.Set("redis/version", redisVersion)
	}

	config.Set("rabbitmq/enabled", configs2.GetOption("rabbitmq/enabled", generalConf, projectConf))
	rabbitVersion := defVersions.RabbitMQ
	if rabbitVersion == "" {
		// Preset didn't pin a RabbitMQ version. Fall back to the
		// current project / global default so we don't blank out a
		// previously-stored version when the user re-runs setup.
		rabbitVersion = configs2.GetOption("rabbitmq/version", generalConf, projectConf)
	}
	repoVersion = strings.Split(rabbitVersion, ":")
	if len(repoVersion) > 1 {
		config.Set("rabbitmq/repository", repoVersion[0])
		config.Set("rabbitmq/version", repoVersion[1])
	} else {
		config.Set("rabbitmq/version", rabbitVersion)
	}
	config.Set("rabbitmq/user", configs2.GetOption("rabbitmq/user", generalConf, projectConf))
	config.Set("rabbitmq/password", configs2.GetOption("rabbitmq/password", generalConf, projectConf))

	// Storefront defaults — Medusa v2 ships Admin UI in the backend
	// (`/app`) but the storefront (browsable shop) is a separate
	// Next.js app at github.com/medusajs/nextjs-starter-medusa.
	// Enable by default; the same nodejs image hosts it on port 8000.
	storefrontEnabled := configs2.GetOption("medusa/storefront/enabled", generalConf, projectConf)
	if storefrontEnabled == "" {
		storefrontEnabled = "true"
	}
	config.Set("medusa/storefront/enabled", storefrontEnabled)

	storefrontRepo := configs2.GetOption("medusa/storefront/repository", generalConf, projectConf)
	if storefrontRepo == "" {
		storefrontRepo = "node"
	}
	config.Set("medusa/storefront/repository", storefrontRepo)

	storefrontVer := configs2.GetOption("medusa/storefront/version", generalConf, projectConf)
	if storefrontVer == "" {
		// Match the backend node image so we don't pull two base
		// layers. defVersions.NodeJs is e.g. "20.18.0".
		storefrontVer = defVersions.NodeJs
	}
	config.Set("medusa/storefront/version", storefrontVer)

	storefrontPath := configs2.GetOption("medusa/storefront/path", generalConf, projectConf)
	if storefrontPath == "" {
		storefrontPath = "storefront"
	}
	config.Set("medusa/storefront/path", storefrontPath)

	storefrontWorkdir := configs2.GetOption("medusa/storefront/workdir", generalConf, projectConf)
	if storefrontWorkdir == "" {
		storefrontWorkdir = "/var/www/html/storefront"
	}
	config.Set("medusa/storefront/workdir", storefrontWorkdir)

	storefrontRegion := configs2.GetOption("medusa/storefront/region", generalConf, projectConf)
	if storefrontRegion == "" {
		// Match the country list seeded by medusa-starter-default's
		// `yarn seed` (gb, de, dk, se, fr, es, it) — first entry.
		storefrontRegion = "gb"
	}
	config.Set("medusa/storefront/region", storefrontRegion)

	storefrontGitURL := configs2.GetOption("medusa/storefront/git_url", generalConf, projectConf)
	if storefrontGitURL == "" {
		storefrontGitURL = "https://github.com/medusajs/nextjs-starter-medusa.git"
	}
	config.Set("medusa/storefront/git_url", storefrontGitURL)

	// Browser-side backend URL for the Next.js storefront. Defaults to
	// the project's first nginx host (https://loc.<project>.com); the
	// user can override it in config.xml when they want a different
	// scheme or domain. SetEnvForProject calls this writer BEFORE it
	// writes `<hosts>` to the project config, so we read the host from
	// `defVersions.Hosts` (populated by the setup wizard) rather than
	// from projectConf, which is still stale at this point.
	publicBackendURL := configs2.GetOption("medusa/storefront/public_backend_url", generalConf, projectConf)
	if publicBackendURL == "" && defVersions.Hosts != "" {
		firstHost := strings.Fields(defVersions.Hosts)[0]
		// Hosts come in as "domain.test:code" pairs (where `code` is
		// the runCode used to namespace nginx/hosts/<code>/name);
		// strip the code suffix so we end up with a plain URL.
		if idx := strings.Index(firstHost, ":"); idx != -1 {
			firstHost = firstHost[:idx]
		}
		publicBackendURL = "https://" + firstHost
	}
	config.Set("medusa/storefront/public_backend_url", publicBackendURL)

	config.Set("grafana/auth/enabled", configs2.GetOption("grafana/auth/enabled", generalConf, projectConf))
	config.Set("grafana/auth/user", configs2.GetOption("grafana/auth/user", generalConf, projectConf))
	config.Set("grafana/auth/password", configs2.GetOption("grafana/auth/password", generalConf, projectConf))
}
