package projects

import (
	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/versions"
	"github.com/faradey/madock/src/versions/magento2"
	"strings"
)

func Shopify(config *configs.ConfigLines, defVersions versions.ToolsVersions, generalConf, projectConfig map[string]string) {
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

	config.AddOrSetLine("NODEJS_ENABLED", "true")
	config.AddOrSetLine("NODEJS_VERSION", defVersions.NodeJs)
	nodeMajorVersion := strings.Split(defVersions.NodeJs, ".")
	if len(nodeMajorVersion) > 0 {
		config.AddOrSetLine("NODEJS_MAJOR_VERSION", nodeMajorVersion[0])
	}

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

	config.AddOrSetLine("YARN_ENABLED", "true")
	config.AddOrSetLine("YARN_VERSION", defVersions.Yarn)
}
