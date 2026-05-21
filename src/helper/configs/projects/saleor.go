package projects

import (
	"strings"

	configs2 "github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/model/versions"
)

func init() {
	RegisterEnvWriter("saleor", Saleor)
}

func Saleor(config *configs2.ConfigLines, defVersions versions.ToolsVersions, generalConf, projectConf map[string]string) {
	if _, ok := projectConf["public_dir"]; !ok {
		config.Set("public_dir", "")
	}
	if _, ok := projectConf["composer_dir"]; !ok {
		config.Set("composer_dir", "")
	}

	config.Set("php/enabled", "false")
	config.Set("php/xdebug/enabled", "false")
	config.Set("nodejs/enabled", "false")
	config.Set("golang/enabled", "false")
	config.Set("ruby/enabled", "false")

	// Saleor backend (uvicorn) listens on port 8000. Make sure the
	// nginx proxy.conf template uses it as the upstream port instead
	// of the default 3000.
	config.Set("main_service_port", "8000")

	config.Set("python/enabled", "true")
	config.Set("python/version", defVersions.Python)
	config.Set("timezone", configs2.GetOption("timezone", generalConf, projectConf))

	// Saleor requires PostgreSQL.
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

	// Saleor uses Redis for the Django cache backend and as the Celery
	// broker (no separate broker needed).
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

	// Optional saleor dashboard container (separate image/repo).
	config.Set("saleor/dashboard/enabled", configs2.GetOption("saleor/dashboard/enabled", generalConf, projectConf))
	config.Set("saleor/dashboard/repository", configs2.GetOption("saleor/dashboard/repository", generalConf, projectConf))
	config.Set("saleor/dashboard/version", configs2.GetOption("saleor/dashboard/version", generalConf, projectConf))

	// Optional celery worker container (same image as the API, separate process).
	config.Set("saleor/worker/enabled", configs2.GetOption("saleor/worker/enabled", generalConf, projectConf))

	config.Set("grafana/auth/enabled", configs2.GetOption("grafana/auth/enabled", generalConf, projectConf))
	config.Set("grafana/auth/user", configs2.GetOption("grafana/auth/user", generalConf, projectConf))
	config.Set("grafana/auth/password", configs2.GetOption("grafana/auth/password", generalConf, projectConf))
}
