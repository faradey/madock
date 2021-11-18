package configs

import (
	"github.com/faradey/madock/src/paths"
	"github.com/faradey/madock/src/versions"
	"log"
	"os"
	"strings"
)

var dbType = "MariaDB"

func SetEnvForProject(defVersions versions.ToolsVersions) {
	projectName := paths.GetRunDirName()
	generalConf := GetGeneralConfig()
	var projectConfig map[string]string
	envFile := paths.MakeDirsByPath(paths.GetExecDirPath()+"/projects/"+projectName) + "/env.txt"
	config := new(ConfigLines)
	config.EnvFile = envFile
	if _, err := os.Stat(envFile); !os.IsNotExist(err) {
		config.IsEnv = true
		projectConfig = GetProjectConfig(projectName)
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

	if !config.IsEnv {
		config.SaveLines()
	}
}

func GetGeneralConfig() map[string]string {
	configPath := paths.GetExecDirPath() + "/projects/config.txt"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err != nil {
			configPath = paths.GetExecDirPath() + "/config.txt"
			if _, err = os.Stat(configPath); os.IsNotExist(err) {
				if err != nil {
					log.Fatal(err)
				}
			}
		}
	}

	return ParseFile(configPath)
}

func GetCurrentProjectConfig() map[string]string {
	return GetProjectConfig(paths.GetRunDirName())
}

func GetProjectConfig(projectName string) map[string]string {
	configPath := paths.GetExecDirPath() + "/projects/" + projectName + "/env.txt"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err != nil {
			log.Fatal(err)
		}
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
