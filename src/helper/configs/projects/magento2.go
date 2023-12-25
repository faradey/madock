package projects

import (
	configs2 "github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/model/versions"
	"github.com/faradey/madock/src/model/versions/magento2"
	"strings"
)

func Magento2(config *configs2.ConfigLines, defVersions versions.ToolsVersions, generalConf, projectConf map[string]string) {
	var dbType = "MariaDB"
	config.Set("PHP_VERSION", defVersions.Php)
	config.Set("PHP_COMPOSER_VERSION", defVersions.Composer)
	config.Set("PHP_TZ", configs2.GetOption("PHP_TZ", generalConf, projectConf))
	if _, ok := projectConf["PUBLIC_DIR"]; !ok {
		config.Set("PUBLIC_DIR", "pub")
	}
	config.Set("XDEBUG_VERSION", magento2.GetXdebugVersion(defVersions.Php))
	config.Set("XDEBUG_REMOTE_HOST", "host.docker.internal")
	config.Set("XDEBUG_IDE_KEY", configs2.GetOption("XDEBUG_IDE_KEY", generalConf, projectConf))
	config.Set("XDEBUG_ENABLED", configs2.GetOption("XDEBUG_ENABLED", generalConf, projectConf))
	config.Set("IONCUBE_ENABLED", configs2.GetOption("IONCUBE_ENABLED", generalConf, projectConf))

	nodeMajorVersion := strings.Split(configs2.GetOption("NODEJS_VERSION", generalConf, projectConf), ".")
	if len(nodeMajorVersion) > 0 {
		config.Set("NODEJS_MAJOR_VERSION", nodeMajorVersion[0])
	}

	repoVersion := strings.Split(defVersions.Db, ":")
	if len(repoVersion) > 1 {
		config.Set("DB_REPOSITORY", repoVersion[0])
		config.Set("DB_VERSION", repoVersion[1])
		config.Set("DB_TYPE", repoVersion[0])
	} else {
		config.Set("DB_VERSION", defVersions.Db)
		config.Set("DB_TYPE", dbType)
	}

	config.Set("DB_ROOT_PASSWORD", configs2.GetOption("DB_ROOT_PASSWORD", generalConf, projectConf))
	config.Set("DB_USER", configs2.GetOption("DB_USER", generalConf, projectConf))
	config.Set("DB_PASSWORD", configs2.GetOption("DB_PASSWORD", generalConf, projectConf))
	config.Set("DB_DATABASE", configs2.GetOption("DB_DATABASE", generalConf, projectConf))

	config.Set("SEARCH_ENGINE", defVersions.SearchEngine)
	if defVersions.SearchEngine == "Elasticsearch" {
		config.Set("OPENSEARCH_ENABLED", "false")
		config.Set("OPENSEARCH_VERSION", defVersions.OpenSearch)

		config.Set("ELASTICSEARCH_ENABLED", "true")
		repoVersion = strings.Split(defVersions.Elastic, ":")
		if len(repoVersion) > 1 {
			config.Set("ELASTICSEARCH_REPOSITORY", repoVersion[0])
			config.Set("ELASTICSEARCH_VERSION", repoVersion[1])
		} else {
			config.Set("ELASTICSEARCH_VERSION", defVersions.Elastic)
		}
	} else if defVersions.SearchEngine == "OpenSearch" {
		config.Set("ELASTICSEARCH_ENABLED", "false")
		config.Set("ELASTICSEARCH_VERSION", defVersions.Elastic)

		config.Set("OPENSEARCH_ENABLED", "true")
		repoVersion = strings.Split(defVersions.OpenSearch, ":")
		if len(repoVersion) > 1 {
			config.Set("OPENSEARCH_REPOSITORY", repoVersion[0])
			config.Set("OPENSEARCH_VERSION", repoVersion[1])
		} else {
			config.Set("OPENSEARCH_VERSION", defVersions.OpenSearch)
		}
	} else {
		config.Set("ELASTICSEARCH_ENABLED", "false")
		config.Set("ELASTICSEARCH_VERSION", defVersions.Elastic)
		config.Set("OPENSEARCH_ENABLED", "false")
		config.Set("OPENSEARCH_VERSION", defVersions.OpenSearch)
	}

	config.Set("REDIS_ENABLED", configs2.GetOption("REDIS_ENABLED", generalConf, projectConf))
	repoVersion = strings.Split(defVersions.Redis, ":")
	if len(repoVersion) > 1 {
		config.Set("REDIS_REPOSITORY", repoVersion[0])
		config.Set("REDIS_VERSION", repoVersion[1])
	} else {
		config.Set("REDIS_VERSION", defVersions.Redis)
	}

	config.Set("NODEJS_ENABLED", configs2.GetOption("NODEJS_ENABLED", generalConf, projectConf))
	config.Set("NODEJS_VERSION", generalConf["NODEJS_VERSION"])

	config.Set("RABBITMQ_ENABLED", configs2.GetOption("RABBITMQ_ENABLED", generalConf, projectConf))
	repoVersion = strings.Split(defVersions.RabbitMQ, ":")
	if len(repoVersion) > 1 {
		config.Set("RABBITMQ_REPOSITORY", repoVersion[0])
		config.Set("RABBITMQ_VERSION", repoVersion[1])
	} else {
		config.Set("RABBITMQ_VERSION", defVersions.RabbitMQ)
	}
}
