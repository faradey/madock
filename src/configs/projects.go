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
	envFile := paths.GetExecDirPath() + "/projects/" + projectName + "/env"
	config := new(ConfigLines)
	config.EnvFile = envFile
	config.AddLine("PHP_VERSION", defVersions.Php)
	config.AddLine("PHP_COMPOSER_VERSION", defVersions.Composer)
	config.AddLine("PHP_TZ", generalConf["PHP_TZ"])
	config.AddLine("PHP_XDEBUG_REMOTE_HOST", "host.docker.internal")
	config.AddLine("PHP_XDEBUG_IDE_KEY", generalConf["PHP_XDEBUG_IDE_KEY"])
	config.AddLine("PHP_MODULE_XDEBUG", generalConf["PHP_MODULE_XDEBUG"])
	config.AddLine("PHP_MODULE_IONCUBE", generalConf["PHP_MODULE_IONCUBE"])

	config.AddEmptyLine()

	config.AddLine("DB_VERSION", defVersions.Db)
	config.AddLine("DB_TYPE", dbType)
	config.AddLine("DB_ROOT_PASSWORD", generalConf["DB_ROOT_PASSWORD"])
	config.AddLine("DB_USER", generalConf["DB_USER"])
	config.AddLine("DB_PASSWORD", generalConf["DB_PASSWORD"])
	config.AddLine("DB_DATABASE", generalConf["DB_DATABASE"])

	config.AddEmptyLine()

	config.AddLine("ELASTICSEARCH_ENABLE", generalConf["ELASTICSEARCH_ENABLE"])
	config.AddLine("ELASTICSEARCH_VERSION", defVersions.Elastic)

	config.AddEmptyLine()

	config.AddLine("REDIS_ENABLE", generalConf["REDIS_ENABLE"])
	config.AddLine("REDIS_VERSION", defVersions.Redis)

	config.AddEmptyLine()

	config.AddLine("RABBITMQ_ENABLE", generalConf["RABBITMQ_ENABLE"])
	config.AddLine("RABBITMQ_VERSION", defVersions.RabbitMQ)

	config.AddEmptyLine()

	config.AddLine("CRON_ENABLED", generalConf["CRON_ENABLED"])

	config.SaveLines()
}

func GetGeneralConfig() map[string]string {
	configPath := paths.GetExecDirPath() + "/projects/config"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err != nil {
			configPath = paths.GetExecDirPath() + "/projects/config.def"
			if _, err = os.Stat(configPath); os.IsNotExist(err) {
				if err != nil {
					log.Fatal(err)
				}
			}
		}
	}

	return ParseFile(configPath)
}

func GetProjectConfig() map[string]string {
	configPath := paths.GetExecDirPath() + "/projects/" + paths.GetRunDirName() + "/env"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err != nil {
			log.Fatal(err)
		}
	}

	config := ParseFile(configPath)
	ConfigMapping(GetGeneralConfig(), config)

	return config
}
