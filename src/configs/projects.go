package configs

import (
	"github.com/faradey/madock/src/paths"
	"github.com/faradey/madock/src/versions"
	"log"
	"os"
)

var dbType = "MariaDB"

func SetEnvForProject(defVersions versions.ToolsVersions) {
	projectName := paths.GetRunDirName()
	generalConf := GetGeneralConfig()
	envFile := paths.MakeDirsByPath(paths.GetExecDirPath()+"/projects/"+projectName) + "/env.txt"
	config := new(ConfigLines)
	config.EnvFile = envFile
	if _, err := os.Stat(envFile); !os.IsNotExist(err) {
		config.IsEnv = true
	}

	config.AddOrSetLine("PHP_VERSION", defVersions.Php)
	config.AddOrSetLine("PHP_COMPOSER_VERSION", defVersions.Composer)
	config.AddOrSetLine("PHP_TZ", generalConf["PHP_TZ"])
	config.AddOrSetLine("PHP_XDEBUG_VERSION", defVersions.Xdebug)
	config.AddOrSetLine("PHP_XDEBUG_REMOTE_HOST", "host.docker.internal")
	config.AddOrSetLine("PHP_XDEBUG_IDE_KEY", generalConf["PHP_XDEBUG_IDE_KEY"])
	config.AddOrSetLine("PHP_MODULE_XDEBUG", generalConf["PHP_MODULE_XDEBUG"])
	config.AddOrSetLine("PHP_MODULE_IONCUBE", generalConf["PHP_MODULE_IONCUBE"])

	if !config.IsEnv {
		config.AddEmptyLine()
	}

	config.AddOrSetLine("DB_VERSION", defVersions.Db)
	config.AddOrSetLine("DB_TYPE", dbType)
	config.AddOrSetLine("DB_ROOT_PASSWORD", generalConf["DB_ROOT_PASSWORD"])
	config.AddOrSetLine("DB_USER", generalConf["DB_USER"])
	config.AddOrSetLine("DB_PASSWORD", generalConf["DB_PASSWORD"])
	config.AddOrSetLine("DB_DATABASE", generalConf["DB_DATABASE"])

	if !config.IsEnv {
		config.AddEmptyLine()
	}

	config.AddOrSetLine("ELASTICSEARCH_ENABLE", generalConf["ELASTICSEARCH_ENABLE"])
	config.AddOrSetLine("ELASTICSEARCH_VERSION", defVersions.Elastic)

	if !config.IsEnv {
		config.AddEmptyLine()
	}

	config.AddOrSetLine("REDIS_ENABLE", generalConf["REDIS_ENABLE"])
	config.AddOrSetLine("REDIS_VERSION", defVersions.Redis)

	if !config.IsEnv {
		config.AddEmptyLine()
	}

	config.AddOrSetLine("NODEJS_ENABLE", generalConf["NODEJS_ENABLE"])
	config.AddOrSetLine("NODEJS_VERSION", generalConf["NODEJS_VERSION"])

	if !config.IsEnv {
		config.AddEmptyLine()
	}

	config.AddOrSetLine("RABBITMQ_ENABLE", generalConf["RABBITMQ_ENABLE"])
	config.AddOrSetLine("RABBITMQ_VERSION", defVersions.RabbitMQ)

	if !config.IsEnv {
		config.AddEmptyLine()
	}

	config.AddOrSetLine("CRON_ENABLED", generalConf["CRON_ENABLED"])

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
