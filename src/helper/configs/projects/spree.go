package projects

import (
	"strings"

	configs2 "github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/model/versions"
)

func init() {
	RegisterEnvWriter("spree", Spree)
}

func Spree(config *configs2.ConfigLines, defVersions versions.ToolsVersions, generalConf, projectConf map[string]string) {
	if _, ok := projectConf["public_dir"]; !ok {
		config.Set("public_dir", "")
	}
	if _, ok := projectConf["composer_dir"]; !ok {
		config.Set("composer_dir", "")
	}

	config.Set("php/enabled", "false")
	config.Set("php/xdebug/enabled", "false")
	config.Set("nodejs/enabled", "false")
	config.Set("python/enabled", "false")
	config.Set("golang/enabled", "false")

	// Spree (Rails) listens on port 3000 by default. proxy.conf uses
	// {{{main_service_port}}} so we keep the rails convention.
	config.Set("main_service_port", "3000")

	config.Set("ruby/enabled", "true")
	config.Set("ruby/version", defVersions.Ruby)
	config.Set("timezone", configs2.GetOption("timezone", generalConf, projectConf))

	// Spree starter ships with PostgreSQL out of the box. MySQL/MariaDB
	// would technically work via Active Record, but the upstream
	// Dockerfile / db migrations target postgres.
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

	// Spree uses Redis for ActiveJob (Sidekiq backend) and Rails cache.
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

	// Optional Sidekiq worker container reuses the ruby image and runs
	// `bundle exec sidekiq` against the same DB/Redis.
	config.Set("spree/sidekiq/enabled", configs2.GetOption("spree/sidekiq/enabled", generalConf, projectConf))

	// Storefront defaults — Spree v5 ships admin in the Rails app and
	// recommends the standalone Next.js storefront at
	// github.com/spree/storefront. Enable by default; the storefront
	// container hosts it on port 3001.
	storefrontEnabled := configs2.GetOption("spree/storefront/enabled", generalConf, projectConf)
	if storefrontEnabled == "" {
		storefrontEnabled = "true"
	}
	config.Set("spree/storefront/enabled", storefrontEnabled)

	storefrontRepo := configs2.GetOption("spree/storefront/repository", generalConf, projectConf)
	if storefrontRepo == "" {
		storefrontRepo = "node"
	}
	config.Set("spree/storefront/repository", storefrontRepo)

	storefrontVer := configs2.GetOption("spree/storefront/version", generalConf, projectConf)
	if storefrontVer == "" {
		// Spree storefront's @inquirer/confirm transitive dep requires
		// Node 22.13+; older Node 22.x patch releases fail at yarn
		// install. Pin to a release that satisfies the constraint.
		storefrontVer = "22.20.0"
	}
	config.Set("spree/storefront/version", storefrontVer)

	storefrontPath := configs2.GetOption("spree/storefront/path", generalConf, projectConf)
	if storefrontPath == "" {
		storefrontPath = "storefront"
	}
	config.Set("spree/storefront/path", storefrontPath)

	storefrontWorkdir := configs2.GetOption("spree/storefront/workdir", generalConf, projectConf)
	if storefrontWorkdir == "" {
		storefrontWorkdir = "/var/www/html/storefront"
	}
	config.Set("spree/storefront/workdir", storefrontWorkdir)

	storefrontCountry := configs2.GetOption("spree/storefront/country", generalConf, projectConf)
	if storefrontCountry == "" {
		storefrontCountry = "us"
	}
	config.Set("spree/storefront/country", storefrontCountry)

	storefrontLocale := configs2.GetOption("spree/storefront/locale", generalConf, projectConf)
	if storefrontLocale == "" {
		storefrontLocale = "en"
	}
	config.Set("spree/storefront/locale", storefrontLocale)

	storefrontStoreName := configs2.GetOption("spree/storefront/store_name", generalConf, projectConf)
	if storefrontStoreName == "" {
		storefrontStoreName = "Spree Store"
	}
	config.Set("spree/storefront/store_name", storefrontStoreName)

	storefrontGitURL := configs2.GetOption("spree/storefront/git_url", generalConf, projectConf)
	if storefrontGitURL == "" {
		storefrontGitURL = "https://github.com/spree/storefront.git"
	}
	config.Set("spree/storefront/git_url", storefrontGitURL)

	// Public-facing storefront URL — browser uses it for SEO meta tags
	// and Open Graph data. Defaults to the project's first nginx host
	// (https://loc.<project>.com); the user can override in config.xml.
	// SetEnvForProject writes hosts AFTER this writer runs, so read
	// from defVersions.Hosts (just collected by the wizard).
	storefrontSiteURL := configs2.GetOption("spree/storefront/site_url", generalConf, projectConf)
	if storefrontSiteURL == "" && defVersions.Hosts != "" {
		firstHost := strings.Fields(defVersions.Hosts)[0]
		if idx := strings.Index(firstHost, ":"); idx != -1 {
			firstHost = firstHost[:idx]
		}
		storefrontSiteURL = "https://" + firstHost
	}
	config.Set("spree/storefront/site_url", storefrontSiteURL)

	config.Set("grafana/auth/enabled", configs2.GetOption("grafana/auth/enabled", generalConf, projectConf))
	config.Set("grafana/auth/user", configs2.GetOption("grafana/auth/user", generalConf, projectConf))
	config.Set("grafana/auth/password", configs2.GetOption("grafana/auth/password", generalConf, projectConf))
}
