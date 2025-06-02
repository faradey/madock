package main

import (
	"github.com/faradey/madock/src/controller/general/bash"
	"github.com/faradey/madock/src/controller/general/clean_cache"
	"github.com/faradey/madock/src/controller/general/cli"
	"github.com/faradey/madock/src/controller/general/composer"
	"github.com/faradey/madock/src/controller/general/config"
	"github.com/faradey/madock/src/controller/general/cron"
	"github.com/faradey/madock/src/controller/general/db/export"
	"github.com/faradey/madock/src/controller/general/db/import"
	info2 "github.com/faradey/madock/src/controller/general/db/info"
	"github.com/faradey/madock/src/controller/general/debug"
	"github.com/faradey/madock/src/controller/general/help"
	"github.com/faradey/madock/src/controller/general/info"
	"github.com/faradey/madock/src/controller/general/install"
	"github.com/faradey/madock/src/controller/general/isnotdefine"
	"github.com/faradey/madock/src/controller/general/logs"
	"github.com/faradey/madock/src/controller/general/node"
	"github.com/faradey/madock/src/controller/general/open"
	"github.com/faradey/madock/src/controller/general/patch"
	"github.com/faradey/madock/src/controller/general/project/clone"
	"github.com/faradey/madock/src/controller/general/project/remove"
	"github.com/faradey/madock/src/controller/general/proxy"
	"github.com/faradey/madock/src/controller/general/prune"
	"github.com/faradey/madock/src/controller/general/rebuild"
	db2 "github.com/faradey/madock/src/controller/general/remote_sync/db"
	"github.com/faradey/madock/src/controller/general/remote_sync/file"
	"github.com/faradey/madock/src/controller/general/remote_sync/media"
	"github.com/faradey/madock/src/controller/general/restart"
	"github.com/faradey/madock/src/controller/general/scope/add"
	listScope "github.com/faradey/madock/src/controller/general/scope/list"
	"github.com/faradey/madock/src/controller/general/scope/set"
	"github.com/faradey/madock/src/controller/general/service/disable"
	"github.com/faradey/madock/src/controller/general/service/enable"
	"github.com/faradey/madock/src/controller/general/service/list"
	"github.com/faradey/madock/src/controller/general/setup"
	"github.com/faradey/madock/src/controller/general/setup/env"
	"github.com/faradey/madock/src/controller/general/snapshot/create"
	"github.com/faradey/madock/src/controller/general/snapshot/restore"
	"github.com/faradey/madock/src/controller/general/ssl"
	"github.com/faradey/madock/src/controller/general/start"
	"github.com/faradey/madock/src/controller/general/status"
	"github.com/faradey/madock/src/controller/general/stop"
	"github.com/faradey/madock/src/controller/magento"
	"github.com/faradey/madock/src/controller/magento/cloud"
	"github.com/faradey/madock/src/controller/magento/mftf"
	"github.com/faradey/madock/src/controller/magento/n98"
	"github.com/faradey/madock/src/controller/prestashop"
	"github.com/faradey/madock/src/controller/pwa"
	"github.com/faradey/madock/src/controller/shopify"
	"github.com/faradey/madock/src/controller/shopify/frontend"
	"github.com/faradey/madock/src/controller/shopify/web"
	"github.com/faradey/madock/src/controller/shopware"
	"github.com/faradey/madock/src/helper/compress"
	"github.com/faradey/madock/src/migration"
	"log"
	"os"
	"strings"
)

var appVersion = "2.9.1"

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.Lmicroseconds)
	migration.Apply(appVersion)
	if len(os.Args) <= 1 {
		help.Execute()
		return
	}

	command := strings.ToLower(os.Args[1])

	switch command {
	case "bash":
		bash.Execute()
	case "c:f":
		clean_cache.Execute()
	case "magento-cloud", "cloud":
		cloud.Execute()
	case "cli":
		cli.Execute()
	case "composer":
		composer.Execute()
	case "compress":
		compress.Zip()
	case "config:cache:clean", "c:c:c":
		config.CacheClean()
	case "config:list":
		config.ShowEnv()
	case "config:set":
		config.SetEnvOption()
	case "cron:enable":
		cron.Enable()
	case "cron:disable":
		cron.Disable()
	case "db:import":
		_import.Import()
	case "db:export":
		export.Export()
	case "db:info":
		info2.Info()
	case "debug:enable":
		debug.Enable()
	case "debug:disable":
		debug.Disable()
	case "debug:profile:enable":
		debug.ProfileEnable()
	case "debug:profile:disable":
		debug.ProfileDisable()
	case "info":
		info.Info()
	case "install":
		install.Execute()
	case "help":
		help.Execute()
	case "logs":
		logs.Execute()
	case "magento", "m":
		magento.Execute()
	case "mftf":
		mftf.Execute()
	case "mftf:init":
		mftf.Init()
	case "n98":
		n98.Execute()
	case "node":
		node.Execute()
	case "open":
		open.Execute()
	case "patch:create":
		patch.Execute()
	case "prestashop", "ps":
		prestashop.Execute()
	case "project:clone":
		clone.Execute()
	case "project:remove":
		remove.Execute()
	case "proxy:start":
		proxy.Execute("start")
	case "proxy:stop":
		proxy.Execute("stop")
	case "proxy:restart":
		proxy.Execute("restart")
	case "proxy:rebuild":
		proxy.Execute("rebuild")
	case "proxy:prune":
		proxy.Execute("prune")
	case "prune":
		prune.Execute()
	case "rebuild":
		rebuild.Execute()
	case "remote:sync:db":
		db2.Execute()
	case "remote:sync:media":
		media.Execute()
	case "remote:sync:file":
		file.Execute()
	case "restart":
		restart.Execute()
	case "pwa":
		pwa.Execute()
	case "scope:add":
		add.Execute()
	case "scope:list":
		listScope.Execute()
	case "scope:set":
		set.Execute()
	case "service:list":
		list.Execute()
	case "service:enable":
		enable.Execute()
	case "service:disable":
		disable.Execute()
	case "setup":
		setup.Execute()
	case "setup:env":
		env.Execute()
	case "shopify", "sy":
		shopify.Execute()
	case "shopify:web", "sy:w":
		web.Execute()
	case "shopify:web:frontend", "sy:w:f":
		frontend.Execute()
	case "shopware", "sw":
		shopware.Execute()
	case "snapshot:create":
		create.Execute()
	case "snapshot:restore":
		restore.Execute()
	case "ssl:rebuild":
		ssl.Execute()
	case "start":
		start.Execute()
	case "status":
		status.Execute()
	case "stop":
		stop.Execute()
	case "uncompress":
		compress.Unzip()
	default:
		isnotdefine.Execute(command)
	}
}

//TODO check rabbitMQ in browser
//TODO add new argument "--name, -n" to "madock project:remove" CLI command
