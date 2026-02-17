package projects

import (
	configs2 "github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/model/versions"
	"strings"
)

func init() {
	RegisterEnvWriter("shopware", Shopware)
}

func Shopware(config *configs2.ConfigLines, defVersions versions.ToolsVersions, generalConf, projectConf map[string]string) {
	if _, ok := projectConf["public_dir"]; !ok {
		config.Set("public_dir", "public")
	}

	if _, ok := projectConf["composer_dir"]; !ok {
		config.Set("composer_dir", "")
	}

	config.Set("php/enabled", "true")
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

	config.Set("search/engine", defVersions.SearchEngine)
	if defVersions.SearchEngine == "Elasticsearch" {
		config.Set("search/opensearch/enabled", "false")
		config.Set("search/opensearch/version", defVersions.OpenSearch)

		config.Set("search/elasticsearch/enabled", "true")
		repoVersion = strings.Split(defVersions.Elastic, ":")
		if len(repoVersion) > 1 {
			config.Set("search/elasticsearch/repository", repoVersion[0])
			config.Set("search/elasticsearch/version", repoVersion[1])
		} else {
			config.Set("search/elasticsearch/version", defVersions.Elastic)
		}
	} else if defVersions.SearchEngine == "OpenSearch" {
		config.Set("search/elasticsearch/enabled", "false")
		config.Set("search/elasticsearch/version", defVersions.Elastic)
		config.Set("search/elasticsearch/version", defVersions.Elastic)

		config.Set("search/opensearch/enabled", "true")
		repoVersion = strings.Split(defVersions.OpenSearch, ":")
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

	config.Set("redis/enabled", configs2.GetOption("redis/enabled", generalConf, projectConf))
	repoVersion = strings.Split(defVersions.Redis, ":")
	if len(repoVersion) > 1 {
		config.Set("redis/repository", repoVersion[0])
		config.Set("redis/version", repoVersion[1])
	} else {
		config.Set("redis/version", defVersions.Redis)
	}

	config.Set("nodejs/enabled", configs2.GetOption("nodejs/enabled", generalConf, projectConf))
	config.Set("nodejs/version", generalConf["nodejs/version"])

	config.Set("rabbitmq/enabled", configs2.GetOption("rabbitmq/enabled", generalConf, projectConf))
	repoVersion = strings.Split(defVersions.RabbitMQ, ":")
	if len(repoVersion) > 1 {
		config.Set("rabbitmq/repository", repoVersion[0])
		config.Set("rabbitmq/version", repoVersion[1])
	} else {
		config.Set("rabbitmq/version", defVersions.RabbitMQ)
	}
}
