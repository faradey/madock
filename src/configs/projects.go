package configs

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/faradey/madock/src/paths"
	"github.com/faradey/madock/src/versions"
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

	config.AddOrSetLine("PHP_VERSION", defVersions.Php)
	config.AddOrSetLine("PHP_COMPOSER_VERSION", defVersions.Composer)
	config.AddOrSetLine("PHP_TZ", getOption("PHP_TZ", generalConf, projectConfig))
	config.AddOrSetLine("XDEBUG_VERSION", defVersions.Xdebug)
	config.AddOrSetLine("XDEBUG_REMOTE_HOST", "host.docker.internal")
	config.AddOrSetLine("XDEBUG_IDE_KEY", getOption("XDEBUG_IDE_KEY", generalConf, projectConfig))
	config.AddOrSetLine("XDEBUG_ENABLED", getOption("XDEBUG_ENABLED", generalConf, projectConfig))
	config.AddOrSetLine("IONCUBE_ENABLED", getOption("IONCUBE_ENABLED", generalConf, projectConfig))

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

	config.AddOrSetLine("DB_ROOT_PASSWORD", getOption("DB_ROOT_PASSWORD", generalConf, projectConfig))
	config.AddOrSetLine("DB_USER", getOption("DB_USER", generalConf, projectConfig))
	config.AddOrSetLine("DB_PASSWORD", getOption("DB_PASSWORD", generalConf, projectConfig))
	config.AddOrSetLine("DB_DATABASE", getOption("DB_DATABASE", generalConf, projectConfig))

	if !config.IsEnv {
		config.AddEmptyLine()
	}

	config.AddOrSetLine("ELASTICSEARCH_ENABLED", getOption("ELASTICSEARCH_ENABLED", generalConf, projectConfig))
	repoVersion = strings.Split(defVersions.Elastic, ":")
	if len(repoVersion) > 1 {
		config.AddOrSetLine("ELASTICSEARCH_REPOSITORY", repoVersion[0])
		config.AddOrSetLine("ELASTICSEARCH_VERSION", repoVersion[1])
	} else {
		config.AddOrSetLine("ELASTICSEARCH_VERSION", defVersions.Elastic)
	}

	if !config.IsEnv {
		config.AddEmptyLine()
	}

	config.AddOrSetLine("REDIS_ENABLED", getOption("REDIS_ENABLED", generalConf, projectConfig))
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

	config.AddOrSetLine("NODEJS_ENABLED", getOption("NODEJS_ENABLED", generalConf, projectConfig))
	config.AddOrSetLine("NODEJS_VERSION", generalConf["NODEJS_VERSION"])

	if !config.IsEnv {
		config.AddEmptyLine()
	}

	config.AddOrSetLine("RABBITMQ_ENABLED", getOption("RABBITMQ_ENABLED", generalConf, projectConfig))
	repoVersion = strings.Split(defVersions.RabbitMQ, ":")
	if len(repoVersion) > 1 {
		config.AddOrSetLine("RABBITMQ_REPOSITORY", repoVersion[0])
		config.AddOrSetLine("RABBITMQ_VERSION", repoVersion[1])
	} else {
		config.AddOrSetLine("RABBITMQ_VERSION", defVersions.RabbitMQ)
	}

	if !config.IsEnv {
		config.AddEmptyLine()
	}

	config.AddOrSetLine("CRON_ENABLED", getOption("CRON_ENABLED", generalConf, projectConfig))

	if !config.IsEnv {
		config.AddEmptyLine()
	}

	config.AddOrSetLine("HOSTS", defVersions.Hosts)

	if !config.IsEnv {
		config.AddEmptyLine()
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
	generalConfig := make(map[string]string)
	if _, err := os.Stat(configPath); !os.IsNotExist(err) && err == nil {
		generalConfig = ParseFile(configPath)
	}

	configPath = paths.GetExecDirPath() + "/config.txt"
	origGeneralConfig := make(map[string]string)
	if _, err := os.Stat(configPath); !os.IsNotExist(err) && err == nil {
		origGeneralConfig = ParseFile(configPath)
	}
	GeneralConfigMapping(origGeneralConfig, generalConfig)

	return generalConfig
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
	if val, ok := projectConfig[name]; ok && val != "" {
		return strings.TrimSpace(val)
	}

	if val, ok := generalConfig[name]; ok && val != "" {
		return strings.TrimSpace(generalConfig[name])
	}
	return ""
}

func PrepareDirsForProject() {
	projectName := GetProjectName()
	projectPath := paths.GetExecDirPath() + "/projects/" + projectName
	paths.MakeDirsByPath(projectPath)
	paths.MakeDirsByPath(projectPath + "/docker")
	paths.MakeDirsByPath(projectPath + "/docker/nginx")
}

func GetProjectName() string {
	suffix := ""
	envFile := ""
	name := ""
	for i := 2; i < 1000; i++ {
		name = paths.GetRunDirName() + suffix
		envFile = paths.GetExecDirPath() + "/projects/" + name + "/env.txt"
		if _, err := os.Stat(envFile); !os.IsNotExist(err) {
			projectConf := GetProjectConfig(name)
			if projectConf["PATH"] != name {
				suffix = "-" + strconv.Itoa(i)
			} else {
				break
			}
		} else {
			break
		}
	}

	return paths.GetRunDirName() + suffix
}
