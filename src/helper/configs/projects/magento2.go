package projects

import (
	configs2 "github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/model/versions"
	"github.com/faradey/madock/src/model/versions/magento2"
	"strings"
)

func Magento2(config *configs2.ConfigLines, defVersions versions.ToolsVersions, generalConf, projectConf map[string]string) {
	var dbType = "MariaDB"
	config.AddOrSetLine("PHP_VERSION", defVersions.Php)
	config.AddOrSetLine("PHP_COMPOSER_VERSION", defVersions.Composer)
	config.AddOrSetLine("PHP_TZ", configs2.GetOption("PHP_TZ", generalConf, projectConf))
	config.AddOrSetLine("XDEBUG_VERSION", magento2.GetXdebugVersion(defVersions.Php))
	config.AddOrSetLine("XDEBUG_REMOTE_HOST", "host.docker.internal")
	config.AddOrSetLine("XDEBUG_IDE_KEY", configs2.GetOption("XDEBUG_IDE_KEY", generalConf, projectConf))
	config.AddOrSetLine("XDEBUG_ENABLED", configs2.GetOption("XDEBUG_ENABLED", generalConf, projectConf))
	config.AddOrSetLine("IONCUBE_ENABLED", configs2.GetOption("IONCUBE_ENABLED", generalConf, projectConf))

	if !config.IsEnv {
		config.AddEmptyLine()
	}

	nodeMajorVersion := strings.Split(configs2.GetOption("NODEJS_VERSION", generalConf, projectConf), ".")
	if len(nodeMajorVersion) > 0 {
		config.AddOrSetLine("NODEJS_MAJOR_VERSION", nodeMajorVersion[0])
	}

	if !config.IsEnv {
		config.AddEmptyLine()
	}

	repoVersion := strings.Split(defVersions.Db, ":")
	if len(repoVersion) > 1 {
		config.AddOrSetLine("DB_REPOSITORY", repoVersion[0])
		config.AddOrSetLine("DB_VERSION", repoVersion[1])
		config.AddOrSetLine("DB_TYPE", repoVersion[0])
	} else {
		config.AddOrSetLine("DB_VERSION", defVersions.Db)
		config.AddOrSetLine("DB_TYPE", dbType)
	}

	config.AddOrSetLine("DB_ROOT_PASSWORD", configs2.GetOption("DB_ROOT_PASSWORD", generalConf, projectConf))
	config.AddOrSetLine("DB_USER", configs2.GetOption("DB_USER", generalConf, projectConf))
	config.AddOrSetLine("DB_PASSWORD", configs2.GetOption("DB_PASSWORD", generalConf, projectConf))
	config.AddOrSetLine("DB_DATABASE", configs2.GetOption("DB_DATABASE", generalConf, projectConf))

	if !config.IsEnv {
		config.AddEmptyLine()
	}
	config.AddOrSetLine("SEARCH_ENGINE", defVersions.SearchEngine)
	if defVersions.SearchEngine == "Elasticsearch" {
		config.AddOrSetLine("OPENSEARCH_ENABLED", "false")
		config.AddOrSetLine("OPENSEARCH_VERSION", defVersions.OpenSearch)

		config.AddOrSetLine("ELASTICSEARCH_ENABLED", "true")
		repoVersion = strings.Split(defVersions.Elastic, ":")
		if len(repoVersion) > 1 {
			config.AddOrSetLine("ELASTICSEARCH_REPOSITORY", repoVersion[0])
			config.AddOrSetLine("ELASTICSEARCH_VERSION", repoVersion[1])
		} else {
			config.AddOrSetLine("ELASTICSEARCH_VERSION", defVersions.Elastic)
		}
	} else if defVersions.SearchEngine == "OpenSearch" {
		config.AddOrSetLine("ELASTICSEARCH_ENABLED", "false")
		config.AddOrSetLine("ELASTICSEARCH_VERSION", defVersions.Elastic)

		config.AddOrSetLine("OPENSEARCH_ENABLED", "true")
		repoVersion = strings.Split(defVersions.OpenSearch, ":")
		if len(repoVersion) > 1 {
			config.AddOrSetLine("OPENSEARCH_REPOSITORY", repoVersion[0])
			config.AddOrSetLine("OPENSEARCH_VERSION", repoVersion[1])
		} else {
			config.AddOrSetLine("OPENSEARCH_VERSION", defVersions.OpenSearch)
		}
	} else {
		config.AddOrSetLine("ELASTICSEARCH_ENABLED", "false")
		config.AddOrSetLine("ELASTICSEARCH_VERSION", defVersions.Elastic)
		config.AddOrSetLine("OPENSEARCH_ENABLED", "false")
		config.AddOrSetLine("OPENSEARCH_VERSION", defVersions.OpenSearch)
	}

	if !config.IsEnv {
		config.AddEmptyLine()
	}

	config.AddOrSetLine("REDIS_ENABLED", configs2.GetOption("REDIS_ENABLED", generalConf, projectConf))
	repoVersion = strings.Split(defVersions.Redis, ":")
	if len(repoVersion) > 1 {
		config.AddOrSetLine("REDIS_REPOSITORY", repoVersion[0])
		config.AddOrSetLine("REDIS_VERSION", repoVersion[1])
	} else {
		config.AddOrSetLine("REDIS_VERSION", defVersions.Redis)
	}

	if !config.IsEnv {
		config.AddEmptyLine()
	}

	config.AddOrSetLine("NODEJS_ENABLED", configs2.GetOption("NODEJS_ENABLED", generalConf, projectConf))
	config.AddOrSetLine("NODEJS_VERSION", generalConf["NODEJS_VERSION"])

	if !config.IsEnv {
		config.AddEmptyLine()
	}

	config.AddOrSetLine("RABBITMQ_ENABLED", configs2.GetOption("RABBITMQ_ENABLED", generalConf, projectConf))
	repoVersion = strings.Split(defVersions.RabbitMQ, ":")
	if len(repoVersion) > 1 {
		config.AddOrSetLine("RABBITMQ_REPOSITORY", repoVersion[0])
		config.AddOrSetLine("RABBITMQ_VERSION", repoVersion[1])
	} else {
		config.AddOrSetLine("RABBITMQ_VERSION", defVersions.RabbitMQ)
	}
}
