package projects

import (
	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/versions"
	"github.com/faradey/madock/src/versions/magento2"
	"strings"
)

func Magento2(config *configs.ConfigLines, defVersions versions.ToolsVersions, generalConf, projectConfig map[string]string) {
	var dbType = "MariaDB"
	config.AddOrSetLine("PHP_VERSION", defVersions.Php)
	config.AddOrSetLine("PHP_COMPOSER_VERSION", defVersions.Composer)
	config.AddOrSetLine("PHP_TZ", configs.GetOption("PHP_TZ", generalConf, projectConfig))
	config.AddOrSetLine("XDEBUG_VERSION", magento2.GetXdebugVersion(defVersions.Php))
	config.AddOrSetLine("XDEBUG_REMOTE_HOST", "host.docker.internal")
	config.AddOrSetLine("XDEBUG_IDE_KEY", configs.GetOption("XDEBUG_IDE_KEY", generalConf, projectConfig))
	config.AddOrSetLine("XDEBUG_ENABLED", configs.GetOption("XDEBUG_ENABLED", generalConf, projectConfig))
	config.AddOrSetLine("IONCUBE_ENABLED", configs.GetOption("IONCUBE_ENABLED", generalConf, projectConfig))

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

	config.AddOrSetLine("DB_ROOT_PASSWORD", configs.GetOption("DB_ROOT_PASSWORD", generalConf, projectConfig))
	config.AddOrSetLine("DB_USER", configs.GetOption("DB_USER", generalConf, projectConfig))
	config.AddOrSetLine("DB_PASSWORD", configs.GetOption("DB_PASSWORD", generalConf, projectConfig))
	config.AddOrSetLine("DB_DATABASE", configs.GetOption("DB_DATABASE", generalConf, projectConfig))

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

	config.AddOrSetLine("REDIS_ENABLED", configs.GetOption("REDIS_ENABLED", generalConf, projectConfig))
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

	config.AddOrSetLine("NODEJS_ENABLED", configs.GetOption("NODEJS_ENABLED", generalConf, projectConfig))
	config.AddOrSetLine("NODEJS_VERSION", generalConf["NODEJS_VERSION"])

	if !config.IsEnv {
		config.AddEmptyLine()
	}

	config.AddOrSetLine("RABBITMQ_ENABLED", configs.GetOption("RABBITMQ_ENABLED", generalConf, projectConfig))
	repoVersion = strings.Split(defVersions.RabbitMQ, ":")
	if len(repoVersion) > 1 {
		config.AddOrSetLine("RABBITMQ_REPOSITORY", repoVersion[0])
		config.AddOrSetLine("RABBITMQ_VERSION", repoVersion[1])
	} else {
		config.AddOrSetLine("RABBITMQ_VERSION", defVersions.RabbitMQ)
	}
}
