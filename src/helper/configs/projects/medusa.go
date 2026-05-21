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

	config.Set("search/elasticsearch/enabled", "false")
	config.Set("search/opensearch/enabled", "false")

	// Medusa uses Redis for events/workflow state.
	config.Set("redis/enabled", "true")
	repoVersion = strings.Split(defVersions.Redis, ":")
	if len(repoVersion) > 1 {
		config.Set("redis/repository", repoVersion[0])
		config.Set("redis/version", repoVersion[1])
	} else {
		config.Set("redis/version", defVersions.Redis)
	}

	config.Set("rabbitmq/enabled", configs2.GetOption("rabbitmq/enabled", generalConf, projectConf))
	repoVersion = strings.Split(defVersions.RabbitMQ, ":")
	if len(repoVersion) > 1 {
		config.Set("rabbitmq/repository", repoVersion[0])
		config.Set("rabbitmq/version", repoVersion[1])
	} else {
		config.Set("rabbitmq/version", defVersions.RabbitMQ)
	}
	config.Set("rabbitmq/user", configs2.GetOption("rabbitmq/user", generalConf, projectConf))
	config.Set("rabbitmq/password", configs2.GetOption("rabbitmq/password", generalConf, projectConf))

	config.Set("medusa/storefront/enabled", configs2.GetOption("medusa/storefront/enabled", generalConf, projectConf))
	config.Set("medusa/storefront/repository", configs2.GetOption("medusa/storefront/repository", generalConf, projectConf))
	config.Set("medusa/storefront/version", configs2.GetOption("medusa/storefront/version", generalConf, projectConf))
	config.Set("medusa/storefront/path", configs2.GetOption("medusa/storefront/path", generalConf, projectConf))
	config.Set("medusa/storefront/workdir", configs2.GetOption("medusa/storefront/workdir", generalConf, projectConf))
	config.Set("medusa/storefront/region", configs2.GetOption("medusa/storefront/region", generalConf, projectConf))

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
		publicBackendURL = "https://" + firstHost
	}
	config.Set("medusa/storefront/public_backend_url", publicBackendURL)

	config.Set("grafana/auth/enabled", configs2.GetOption("grafana/auth/enabled", generalConf, projectConf))
	config.Set("grafana/auth/user", configs2.GetOption("grafana/auth/user", generalConf, projectConf))
	config.Set("grafana/auth/password", configs2.GetOption("grafana/auth/password", generalConf, projectConf))
}
