package main

import (
	"github.com/faradey/madock/src/compress"
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
	"github.com/faradey/madock/src/controller/general/proxy"
	"github.com/faradey/madock/src/controller/general/rebuild"
	"github.com/faradey/madock/src/controller/magento/cloud"
	"github.com/faradey/madock/src/migration"
	"log"
	"os"
	"strings"

	"github.com/faradey/madock/src/cli/attr"
	"github.com/faradey/madock/src/cli/commands"
	cliHelper "github.com/faradey/madock/src/helper"
)

var appVersion string = "2.2.0"

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	migration.Apply(appVersion)
	if len(os.Args) <= 1 {
		help.Execute()
		return
	}

	command := strings.ToLower(os.Args[1])
	attr.ParseAttributes()

	switch command {
	case "bash":
		bash.Bash()
	case "c:f":
		clean_cache.CleanCache()
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
		commands.InstallMagento()
	case "help":
		help.Execute()
	case "logs":
		commands.Logs()
	case "magento", "m":
		commands.Magento(cliHelper.NormalizeCliCommandWithJoin(os.Args[2:]))
	case "mftf":
		commands.Mftf(strings.Join(os.Args[2:], " "))
	case "mftf:init":
		commands.MftfInit()
	case "n98":
		commands.N98(cliHelper.NormalizeCliCommandWithJoin(os.Args[2:]))
	case "node":
		commands.Node(cliHelper.NormalizeCliCommandWithJoin(os.Args[2:]))
	case "patch:create":
		commands.PatchCreate()
	case "project:remove":
		commands.ProjectRemove()
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
		commands.Prune()
	case "rebuild":
		rebuild.Execute()
	case "remote:sync:db":
		commands.RemoteSyncDb()
	case "remote:sync:media":
		commands.RemoteSyncMedia()
	case "remote:sync:file":
		commands.RemoteSyncFile()
	case "restart":
		commands.Restart()
	case "pwa":
		commands.PWA(cliHelper.NormalizeCliCommandWithJoin(os.Args[2:]))
	case "service:list":
		commands.ServiceList()
	case "service:enable":
		commands.ServiceEnable()
	case "service:disable":
		commands.ServiceDisable()
	case "setup":
		commands.Setup()
	case "setup:env":
		commands.SetupEnv()
	case "shopify", "sy":
		commands.Shopify(strings.Join(os.Args[2:], " "))
	case "shopify:web", "sy:w":
		commands.ShopifyWeb(strings.Join(os.Args[2:], " "))
	case "shopify:web:frontend", "sy:w:f":
		commands.ShopifyWebFrontend(strings.Join(os.Args[2:], " "))
	case "ssl:rebuild":
		commands.Ssl()
	case "start":
		commands.Start()
	case "status":
		commands.Status()
	case "stop":
		commands.Stop()
	case "uncompress":
		compress.Unzip()
	default:
		commands.IsNotDefine()
	}
}

//TODO check opensearchdashboard in browser
//TODO check rabbitMQ in browser
//TODO check redis in browser
