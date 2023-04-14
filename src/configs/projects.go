package configs

import (
	"github.com/faradey/madock/src/versions"
	"github.com/faradey/madock/src/versions/magento2"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/faradey/madock/src/paths"
)

var dbType = "MariaDB"

func SetEnvForProject(projectName string, defVersions versions.ToolsVersions, projectConfig map[string]string) {
	generalConf := GetGeneralConfig()
	config := new(ConfigLines)
	envFile := paths.MakeDirsByPath(paths.GetExecDirPath()+"/projects/"+projectName) + "/env.txt"
	config.EnvFile = envFile
	if len(projectConfig) > 0 {
		config.IsEnv = true
	}

	config.AddOrSetLine("PATH", paths.GetRunDirPath())
	config.AddOrSetLine("PLATFORM", defVersions.Platform)
	config.AddOrSetLine("PHP_VERSION", defVersions.Php)
	config.AddOrSetLine("PHP_COMPOSER_VERSION", defVersions.Composer)
	config.AddOrSetLine("PHP_TZ", getOption("PHP_TZ", generalConf, projectConfig))
	config.AddOrSetLine("XDEBUG_VERSION", magento2.GetXdebugVersion(defVersions.Php))
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
	return GetProjectConfig(GetProjectName())
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

func PrepareDirsForProject(projectName string) {
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
			val, ok := projectConf["PATH"]
			if ok && val != paths.GetRunDirPath() {
				suffix = "-" + strconv.Itoa(i)
			} else {
				break
			}
		} else {
			break
		}
	}

	return name
}
