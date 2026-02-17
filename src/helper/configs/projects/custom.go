package projects

import (
	"strings"

	configs2 "github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/model/versions"
)

func Custom(config *configs2.ConfigLines, defVersions versions.ToolsVersions, generalConf, projectConf map[string]string) {
	if _, ok := projectConf["public_dir"]; !ok {
		config.Set("public_dir", "public")
	}

	if _, ok := projectConf["composer_dir"]; !ok {
		config.Set("composer_dir", "")
	}

	language := defVersions.Language
	if language == "" {
		language = "php"
	}

	// Reset all language-specific enabled flags
	config.Set("php/enabled", "false")
	config.Set("php/xdebug/enabled", "false")
	config.Set("nodejs/enabled", "false")
	config.Set("python/enabled", "false")
	config.Set("golang/enabled", "false")
	config.Set("ruby/enabled", "false")
	config.Set("app/enabled", "false")

	switch language {
	case "php":
		config.Set("php/enabled", "true")
		customPhpConfig(config, defVersions, generalConf, projectConf)
	case "nodejs":
		customNodeJsConfig(config, defVersions, generalConf, projectConf)
	case "python":
		config.Set("python/enabled", "true")
		customPythonConfig(config, defVersions, generalConf, projectConf)
	case "golang":
		config.Set("golang/enabled", "true")
		customGolangConfig(config, defVersions, generalConf, projectConf)
	case "ruby":
		config.Set("ruby/enabled", "true")
		customRubyConfig(config, defVersions, generalConf, projectConf)
	case "none":
		config.Set("app/enabled", "true")
		customNoneConfig(config, generalConf, projectConf)
	}

	// Common DB config for all languages
	customDbConfig(config, defVersions, generalConf, projectConf)

	// Search engine config only for PHP
	if language == "php" {
		customSearchConfig(config, defVersions, generalConf, projectConf)
	} else {
		config.Set("search/elasticsearch/enabled", "false")
		config.Set("search/opensearch/enabled", "false")
	}

	// Common services
	config.Set("redis/enabled", configs2.GetOption("redis/enabled", generalConf, projectConf))
	repoVersion := strings.Split(defVersions.Redis, ":")
	if len(repoVersion) > 1 {
		config.Set("redis/repository", repoVersion[0])
		config.Set("redis/version", repoVersion[1])
	} else {
		config.Set("redis/version", defVersions.Redis)
	}

	config.Set("rabbitmq/enabled", configs2.GetOption("rabbitmq/enabled", generalConf, projectConf))
	repoVersion = strings.Split(defVersions.RabbitMQ, ":")
	if len(repoVersion) > 1 {
		config.Set("rabbitmq/repository", repoVersion[0])
		config.Set("rabbitmq/version", repoVersion[1])
	} else {
		config.Set("rabbitmq/version", defVersions.RabbitMQ)
	}
}

func customPhpConfig(config *configs2.ConfigLines, defVersions versions.ToolsVersions, generalConf, projectConf map[string]string) {
	config.Set("php/version", defVersions.Php)
	config.Set("php/composer/version", defVersions.Composer)
	config.Set("timezone", configs2.GetOption("timezone", generalConf, projectConf))

	config.Set("php/xdebug/version", versions.GetXdebugVersion(defVersions.Php))
	config.Set("php/xdebug/remote_host", "host.docker.internal")
	config.Set("php/xdebug/ide_key", configs2.GetOption("php/xdebug/ide_key", generalConf, projectConf))
	config.Set("php/xdebug/enabled", configs2.GetOption("php/xdebug/enabled", generalConf, projectConf))
	config.Set("php/ioncube/enabled", configs2.GetOption("php/ioncube/enabled", generalConf, projectConf))

	nodeMajorVersion := strings.Split(configs2.GetOption("nodejs/version", generalConf, projectConf), ".")
	if len(nodeMajorVersion) > 0 {
		config.Set("nodejs/major_version", nodeMajorVersion[0])
	}

	config.Set("php/nodejs/enabled", configs2.GetOption("php/nodejs/enabled", generalConf, projectConf))
	config.Set("nodejs/version", generalConf["nodejs/version"])
}

func customNodeJsConfig(config *configs2.ConfigLines, defVersions versions.ToolsVersions, generalConf, projectConf map[string]string) {
	config.Set("nodejs/enabled", "true")
	config.Set("nodejs/version", defVersions.NodeJs)
	nodeMajorVersion := strings.Split(defVersions.NodeJs, ".")
	if len(nodeMajorVersion) > 0 {
		config.Set("nodejs/major_version", nodeMajorVersion[0])
	}
	config.Set("nodejs/yarn/version", defVersions.Yarn)
	config.Set("timezone", configs2.GetOption("timezone", generalConf, projectConf))
}

func customPythonConfig(config *configs2.ConfigLines, defVersions versions.ToolsVersions, generalConf, projectConf map[string]string) {
	config.Set("python/version", defVersions.Python)
	config.Set("timezone", configs2.GetOption("timezone", generalConf, projectConf))
}

func customGolangConfig(config *configs2.ConfigLines, defVersions versions.ToolsVersions, generalConf, projectConf map[string]string) {
	config.Set("go/version", defVersions.Golang)
	config.Set("timezone", configs2.GetOption("timezone", generalConf, projectConf))
}

func customRubyConfig(config *configs2.ConfigLines, defVersions versions.ToolsVersions, generalConf, projectConf map[string]string) {
	config.Set("ruby/version", defVersions.Ruby)
	config.Set("timezone", configs2.GetOption("timezone", generalConf, projectConf))
}

func customNoneConfig(config *configs2.ConfigLines, generalConf, projectConf map[string]string) {
	config.Set("timezone", configs2.GetOption("timezone", generalConf, projectConf))
}

func customDbConfig(config *configs2.ConfigLines, defVersions versions.ToolsVersions, generalConf, projectConf map[string]string) {
	repoVersion := strings.Split(defVersions.Db, ":")
	if len(repoVersion) > 1 {
		config.Set("db/repository", repoVersion[0])
		config.Set("db/version", repoVersion[1])
	} else {
		config.Set("db/version", defVersions.Db)
	}

	config.Set("db/root_password", configs2.GetOption("db/root_password", generalConf, projectConf))
	config.Set("db/user", configs2.GetOption("db/user", generalConf, projectConf))
	config.Set("db/password", configs2.GetOption("db/password", generalConf, projectConf))
	config.Set("db/database", configs2.GetOption("db/database", generalConf, projectConf))
}

func customSearchConfig(config *configs2.ConfigLines, defVersions versions.ToolsVersions, generalConf, projectConf map[string]string) {
	config.Set("search/engine", defVersions.SearchEngine)
	if defVersions.SearchEngine == "Elasticsearch" {
		config.Set("search/opensearch/enabled", "false")
		config.Set("search/opensearch/version", defVersions.OpenSearch)

		config.Set("search/elasticsearch/enabled", "true")
		repoVersion := strings.Split(defVersions.Elastic, ":")
		if len(repoVersion) > 1 {
			config.Set("search/elasticsearch/repository", repoVersion[0])
			config.Set("search/elasticsearch/version", repoVersion[1])
		} else {
			config.Set("search/elasticsearch/version", defVersions.Elastic)
		}
	} else if defVersions.SearchEngine == "OpenSearch" {
		config.Set("search/elasticsearch/enabled", "false")
		config.Set("search/elasticsearch/version", defVersions.Elastic)

		config.Set("search/opensearch/enabled", "true")
		repoVersion := strings.Split(defVersions.OpenSearch, ":")
		if len(repoVersion) > 1 {
			config.Set("search/opensearch/repository", repoVersion[0])
			config.Set("search/opensearch/version", repoVersion[1])
		} else {
			config.Set("search/opensearch/version", defVersions.OpenSearch)
		}
	} else {
		config.Set("search/elasticsearch/enabled", "false")
		config.Set("search/elasticsearch/version", defVersions.Elastic)
		config.Set("search/opensearch/enabled", "false")
		config.Set("search/opensearch/version", defVersions.OpenSearch)
	}
}
