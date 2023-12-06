package main

import (
	"github.com/faradey/madock/src/controller/general/bash"
	"github.com/faradey/madock/src/controller/general/clean_cache"
	"github.com/faradey/madock/src/controller/general/cli"
	"github.com/faradey/madock/src/controller/general/composer"
	"github.com/faradey/madock/src/controller/general/config"
	"github.com/faradey/madock/src/controller/general/cron"
	"github.com/faradey/madock/src/controller/general/db"
	"github.com/faradey/madock/src/controller/general/debug"
	"github.com/faradey/madock/src/controller/general/help"
	"github.com/faradey/madock/src/controller/general/info"
	"github.com/faradey/madock/src/controller/general/install"
	"github.com/faradey/madock/src/controller/general/isnotdefine"
	"github.com/faradey/madock/src/controller/general/logs"
	"github.com/faradey/madock/src/controller/general/node"
	"github.com/faradey/madock/src/controller/general/patch"
	"github.com/faradey/madock/src/controller/general/project_remove"
	"github.com/faradey/madock/src/controller/general/proxy"
	"github.com/faradey/madock/src/controller/general/prune"
	"github.com/faradey/madock/src/controller/general/rebuild"
	db2 "github.com/faradey/madock/src/controller/general/remote_sync/db"
	"github.com/faradey/madock/src/controller/general/remote_sync/file"
	"github.com/faradey/madock/src/controller/general/remote_sync/media"
	"github.com/faradey/madock/src/controller/general/restart"
	"github.com/faradey/madock/src/controller/general/service/disable"
	"github.com/faradey/madock/src/controller/general/service/enable"
	"github.com/faradey/madock/src/controller/general/service/list"
	"github.com/faradey/madock/src/controller/general/setup"
	"github.com/faradey/madock/src/controller/general/setup/env"
	"github.com/faradey/madock/src/controller/general/ssl"
	"github.com/faradey/madock/src/controller/general/start"
	"github.com/faradey/madock/src/controller/general/status"
	"github.com/faradey/madock/src/controller/general/stop"
	"github.com/faradey/madock/src/controller/magento"
	"github.com/faradey/madock/src/controller/magento/cloud"
	"github.com/faradey/madock/src/controller/magento/mftf"
	"github.com/faradey/madock/src/controller/magento/n98"
	"github.com/faradey/madock/src/controller/pwa"
	"github.com/faradey/madock/src/controller/shopify"
	"github.com/faradey/madock/src/controller/shopify/frontend"
	"github.com/faradey/madock/src/controller/shopify/web"
	"github.com/faradey/madock/src/helper/compress"
	"github.com/faradey/madock/src/migration"
	"log"
	"os"
	"strings"
)

var appVersion string = "2.3.0"

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	migration.Apply(appVersion)
	if len(os.Args) <= 1 {
		help.Execute()
		return
	}

	command := strings.ToLower(os.Args[1])

	switch command {
	case "bash":
		bash.Bash()
	case "c:f":
		clean_cache.Execute()
	case "magento-cloud", "cloud":
		cloud.Cloud()
	case "cli":
		cli.Cli()
	case "composer":
		composer.Composer()
	case "compress":
		compress.Zip()
	case "config:list":
		config.ShowEnv()
	case "config:set":
		config.SetEnvOption()
	case "cron:enable":
		cron.Enable()
	case "cron:disable":
		cron.Disable()
	case "db:import":
		db.Import()
	case "db:export":
		db.Export()
	case "db:info":
		db.Info()
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
	case "patch:create":
		patch.Create()
	case "project:remove":
		project_remove.Execute()
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
		isnotdefine.Execute()
	}
}

//TODO check opensearchdashboard in browser
//TODO check rabbitMQ in browser
//TODO check redis in browser
