package projects

import (
	"strings"

	configs2 "github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/model/versions"
)

func init() {
	RegisterEnvWriter("sylius", Sylius)
}

func Sylius(config *configs2.ConfigLines, defVersions versions.ToolsVersions, generalConf, projectConf map[string]string) {
	// Sylius Standard ships a `public/` web root served by nginx.
	if _, ok := projectConf["public_dir"]; !ok {
		config.Set("public_dir", "public")
	}
	if _, ok := projectConf["composer_dir"]; !ok {
		config.Set("composer_dir", "")
	}

	config.Set("php/enabled", "true")
	config.Set("php/version", defVersions.Php)
	config.Set("php/composer/version", defVersions.Composer)
	config.Set("timezone", configs2.GetOption("timezone", generalConf, projectConf))

	config.Set("php/xdebug/version", versions.GetXdebugVersion(defVersions.Php))
	config.Set("php/xdebug/remote_host", "host.docker.internal")
	config.Set("php/xdebug/ide_key", configs2.GetOption("php/xdebug/ide_key", generalConf, projectConf))
	config.Set("php/xdebug/enabled", configs2.GetOption("php/xdebug/enabled", generalConf, projectConf))
	config.Set("php/ioncube/enabled", configs2.GetOption("php/ioncube/enabled", generalConf, projectConf))

	// Sylius uses Webpack Encore + Gulp for asset bundling. Node + Yarn
	// must be available in the PHP container so `yarn install &&
	// yarn build` works during install.
	nodeVer := defVersions.NodeJs
	if nodeVer == "" {
		nodeVer = configs2.GetOption("nodejs/version", generalConf, projectConf)
	}
	if nodeVer != "" {
		config.Set("nodejs/version", nodeVer)
		nodeMajorVersion := strings.Split(nodeVer, ".")
		if len(nodeMajorVersion) > 0 {
			config.Set("nodejs/major_version", nodeMajorVersion[0])
		}
	}
	config.Set("php/nodejs/enabled", "true")
	config.Set("php/yarn/enabled", "true")
	yarnVer := defVersions.Yarn
	if yarnVer == "" {
		yarnVer = configs2.GetOption("nodejs/yarn/version", generalConf, projectConf)
	}
	if yarnVer != "" {
		config.Set("nodejs/yarn/version", yarnVer)
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

	// Search engines are off by default. Sylius supports Elasticsearch
	// via the `sylius/elasticsearch-plugin` (or the v2 built-in search
	// when configured). Users opt in with `madock service:enable
	// elasticsearch` / `opensearch` and wire the plugin themselves.
	searchEsEnabled := configs2.GetOption("search/elasticsearch/enabled", generalConf, projectConf)
	if searchEsEnabled == "" {
		searchEsEnabled = "false"
	}
	config.Set("search/elasticsearch/enabled", searchEsEnabled)
	if v := configs2.GetOption("search/elasticsearch/version", generalConf, projectConf); v != "" {
		config.Set("search/elasticsearch/version", v)
	} else {
		config.Set("search/elasticsearch/version", "8.11.4")
	}

	searchOsEnabled := configs2.GetOption("search/opensearch/enabled", generalConf, projectConf)
	if searchOsEnabled == "" {
		searchOsEnabled = "false"
	}
	config.Set("search/opensearch/enabled", searchOsEnabled)
	if v := configs2.GetOption("search/opensearch/version", generalConf, projectConf); v != "" {
		config.Set("search/opensearch/version", v)
	} else {
		config.Set("search/opensearch/version", "2.12.0")
	}

	// Sylius uses Symfony Messenger; redis is optional but recommended
	// for the Doctrine cache + Messenger transport.
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
	} else if defVersions.RabbitMQ != "" {
		config.Set("rabbitmq/version", defVersions.RabbitMQ)
	}
	config.Set("rabbitmq/user", configs2.GetOption("rabbitmq/user", generalConf, projectConf))
	config.Set("rabbitmq/password", configs2.GetOption("rabbitmq/password", generalConf, projectConf))

	// Optional Symfony Messenger consumer container — reuses the PHP
	// image and runs `messenger:consume async` against the shared
	// Doctrine transport. Enable with `madock service:enable messenger`.
	config.Set("sylius/messenger/enabled", configs2.GetOption("sylius/messenger/enabled", generalConf, projectConf))

	// Optional Webpack Encore watcher container — reuses the PHP
	// image and runs `yarn watch` for live rebuilds on assets/ change.
	// Enable with `madock service:enable encore`.
	config.Set("sylius/encore/enabled", configs2.GetOption("sylius/encore/enabled", generalConf, projectConf))

	config.Set("grafana/auth/enabled", configs2.GetOption("grafana/auth/enabled", generalConf, projectConf))
	config.Set("grafana/auth/user", configs2.GetOption("grafana/auth/user", generalConf, projectConf))
	config.Set("grafana/auth/password", configs2.GetOption("grafana/auth/password", generalConf, projectConf))
}
