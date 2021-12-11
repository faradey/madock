package configs

import (
	"github.com/faradey/madock/src/paths"
	"github.com/faradey/madock/src/versions"
	"log"
	"os"
	"strings"
)

var dbType = "MariaDB"

func SetEnvForProject(defVersions versions.ToolsVersions, projectConfig map[string]string) {
	projectName := paths.GetRunDirName()
	generalConf := GetGeneralConfig()
	config := new(ConfigLines)
	envFile := paths.MakeDirsByPath(paths.GetExecDirPath()+"/projects/"+projectName) + "/env.txt"
	config.EnvFile = envFile
	if len(projectConfig) > 0 {
		config.IsEnv = true
	}

	config.AddOrSetLine("NGINX_PLACE", getOption("NGINX_PLACE", generalConf, projectConfig))
	config.AddOrSetLine("PHP_VERSION", defVersions.Php)
	config.AddOrSetLine("PHP_COMPOSER_VERSION", defVersions.Composer)
	config.AddOrSetLine("PHP_TZ", getOption("PHP_TZ", generalConf, projectConfig))
	config.AddOrSetLine("PHP_XDEBUG_VERSION", defVersions.Xdebug)
	config.AddOrSetLine("PHP_XDEBUG_REMOTE_HOST", "host.docker.internal")
	config.AddOrSetLine("PHP_XDEBUG_IDE_KEY", getOption("PHP_XDEBUG_IDE_KEY", generalConf, projectConfig))
	config.AddOrSetLine("PHP_MODULE_XDEBUG", getOption("PHP_MODULE_XDEBUG", generalConf, projectConfig))
	config.AddOrSetLine("PHP_MODULE_IONCUBE", getOption("PHP_MODULE_IONCUBE", generalConf, projectConfig))

	if !config.IsEnv {
		config.AddEmptyLine()
	}

	config.AddOrSetLine("DB_VERSION", defVersions.Db)
	config.AddOrSetLine("DB_TYPE", dbType)
	config.AddOrSetLine("DB_ROOT_PASSWORD", getOption("DB_ROOT_PASSWORD", generalConf, projectConfig))
	config.AddOrSetLine("DB_USER", getOption("DB_USER", generalConf, projectConfig))
	config.AddOrSetLine("DB_PASSWORD", getOption("DB_PASSWORD", generalConf, projectConfig))
	config.AddOrSetLine("DB_DATABASE", getOption("DB_DATABASE", generalConf, projectConfig))

	if !config.IsEnv {
		config.AddEmptyLine()
	}

	config.AddOrSetLine("ELASTICSEARCH_ENABLE", getOption("ELASTICSEARCH_ENABLE", generalConf, projectConfig))
	config.AddOrSetLine("ELASTICSEARCH_VERSION", defVersions.Elastic)

	if !config.IsEnv {
		config.AddEmptyLine()
	}

	config.AddOrSetLine("REDIS_ENABLE", getOption("REDIS_ENABLE", generalConf, projectConfig))
	config.AddOrSetLine("REDIS_VERSION", defVersions.Redis)

	if !config.IsEnv {
		config.AddEmptyLine()
	}

	config.AddOrSetLine("NODEJS_ENABLE", getOption("NODEJS_ENABLE", generalConf, projectConfig))
	config.AddOrSetLine("NODEJS_VERSION", generalConf["NODEJS_VERSION"])

	if !config.IsEnv {
		config.AddEmptyLine()
	}

	config.AddOrSetLine("RABBITMQ_ENABLE", getOption("RABBITMQ_ENABLE", generalConf, projectConfig))
	config.AddOrSetLine("RABBITMQ_VERSION", defVersions.RabbitMQ)

	if !config.IsEnv {
		config.AddEmptyLine()
	}

	config.AddOrSetLine("CRON_ENABLED", getOption("CRON_ENABLED", generalConf, projectConfig))
	if val, ok := projectConfig["HOSTS"]; ok {
		config.AddOrSetLine("HOSTS", val)
	} else {
		config.AddOrSetLine("HOSTS", defVersions.Hosts)
	}

	config.AddOrSetLine("SSH_AUTH_TYPE", getOption("SSH_AUTH_TYPE", generalConf, projectConfig))
	config.AddOrSetLine("SSH_HOST", getOption("SSH_HOST", generalConf, projectConfig))
	config.AddOrSetLine("SSH_PORT", getOption("SSH_PORT", generalConf, projectConfig))
	config.AddOrSetLine("SSH_USERNAME", getOption("SSH_USERNAME", generalConf, projectConfig))
	config.AddOrSetLine("SSH_KEY_PATH", getOption("SSH_KEY_PATH", generalConf, projectConfig))
	config.AddOrSetLine("SSH_PASSWORD", getOption("SSH_PASSWORD", generalConf, projectConfig))
	config.AddOrSetLine("SSH_SITE_ROOT_PATH", getOption("SSH_SITE_ROOT_PATH", generalConf, projectConfig))

	if !config.IsEnv {
		config.SaveLines()
	}
}

func GetGeneralConfig() map[string]string {
	configPath := paths.GetExecDirPath() + "/projects/config.txt"
	if _, err := os.Stat(configPath); os.IsNotExist(err) && err != nil {
		configPath = paths.GetExecDirPath() + "/config.txt"
		if _, err = os.Stat(configPath); os.IsNotExist(err) && err != nil {
			log.Fatal(err)
		}
	}

	return ParseFile(configPath)
}

func GetCurrentProjectConfig() map[string]string {
	return GetProjectConfig(paths.GetRunDirName())
}

func GetProjectConfig(projectName string) map[string]string {
	configPath := paths.GetExecDirPath() + "/projects/" + projectName + "/env.txt"
	if _, err := os.Stat(configPath); os.IsNotExist(err) && err != nil {
		log.Fatal(err)
	}

	config := ParseFile(configPath)
	ConfigMapping(GetGeneralConfig(), config)

	return config
}

func getOption(name string, generalConfig, projectConfig map[string]string) string {
	if val, ok := projectConfig[name]; ok {
		return strings.TrimSpace(val)
	}

	return strings.TrimSpace(generalConfig[name])
}
