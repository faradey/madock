package projects

import (
	"strings"

	configs2 "github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/model/versions"
)

func init() {
	RegisterEnvWriter("shopify", Shopify)
}

func Shopify(config *configs2.ConfigLines, defVersions versions.ToolsVersions, generalConf, projectConf map[string]string) {
	config.Set("php/enabled", "true")
	config.Set("php/version", defVersions.Php)
	config.Set("php/composer/version", defVersions.Composer)
	config.Set("timezone", configs2.GetOption("timezone", generalConf, projectConf))

	if _, ok := projectConf["public_dir"]; !ok {
		config.Set("public_dir", "web/public")
	}

	if _, ok := projectConf["composer_dir"]; !ok {
		config.Set("composer_dir", "web")
	}

	config.Set("php/xdebug/version", versions.GetXdebugVersion(defVersions.Php))
	config.Set("php/xdebug/remote_host", "host.docker.internal")
	config.Set("php/xdebug/ide_key", configs2.GetOption("php/xdebug/ide_key", generalConf, projectConf))
	config.Set("php/xdebug/enabled", configs2.GetOption("php/xdebug/enabled", generalConf, projectConf))
	config.Set("php/ioncube/enabled", configs2.GetOption("php/ioncube/enabled", generalConf, projectConf))

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

	config.Set("redis/enabled", configs2.GetOption("redis/enabled", generalConf, projectConf))
	repoVersion = strings.Split(defVersions.Redis, ":")
	if len(repoVersion) > 1 {
		config.Set("redis/repository", repoVersion[0])
		config.Set("redis/version", repoVersion[1])
	} else {
		config.Set("redis/version", defVersions.Redis)
	}

	config.Set("nodejs/enabled", "true")
	config.Set("nodejs/version", defVersions.NodeJs)
	nodeMajorVersion := strings.Split(defVersions.NodeJs, ".")
	if len(nodeMajorVersion) > 0 {
		config.Set("nodejs/major_version", nodeMajorVersion[0])
	}

	config.Set("rabbitmq/enabled", configs2.GetOption("rabbitmq/enabled", generalConf, projectConf))
	repoVersion = strings.Split(defVersions.RabbitMQ, ":")
	if len(repoVersion) > 1 {
		config.Set("rabbitmq/repository", repoVersion[0])
		config.Set("rabbitmq/version", repoVersion[1])
	} else {
		config.Set("rabbitmq/version", defVersions.RabbitMQ)
	}

	config.Set("nodejs/yarn/enabled", "true")
	config.Set("nodejs/yarn/version", defVersions.Yarn)
}
