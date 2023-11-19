package main

import (
	"github.com/faradey/madock/src/controller/def/bash"
	"github.com/faradey/madock/src/controller/def/clean_cache"
	"github.com/faradey/madock/src/migration"
	"log"
	"os"
	"strings"

	"github.com/faradey/madock/src/cli/attr"
	"github.com/faradey/madock/src/cli/commands"
	"github.com/faradey/madock/src/cli/helper"
	cliHelper "github.com/faradey/madock/src/helper"
)

var appVersion string = "2.2.0"

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	migration.Apply(appVersion)
	if len(os.Args) <= 1 {
		helper.Help()
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
		commands.Cloud(cliHelper.NormalizeCliCommandWithJoin(os.Args[2:]))
	case "cli":
		commands.Cli(cliHelper.NormalizeCliCommandWithJoin(os.Args[2:]))
	case "composer":
		commands.Composer(cliHelper.NormalizeCliCommandWithJoin(os.Args[2:]))
	case "compress":
		commands.Compress()
	case "config:list":
		commands.ShowEnv()
	case "config:set":
		commands.SetEnvOption()
	case "cron:enable":
		commands.CronEnable()
	case "cron:disable":
		commands.CronDisable()
	case "db:import":
		commands.DBImport()
	case "db:export":
		commands.DBExport()
	case "db:info":
		commands.DBInfo()
	case "debug:enable":
		commands.DebugEnable()
	case "debug:disable":
		commands.DebugDisable()
	case "debug:profile:enable":
		commands.DebugProfileEnable()
	case "debug:profile:disable":
		commands.DebugProfileDisable()
	case "info":
		commands.Info()
	case "install":
		commands.InstallMagento()
	case "help":
		helper.Help()
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
		commands.Proxy("start")
	case "proxy:stop":
		commands.Proxy("stop")
	case "proxy:restart":
		commands.Proxy("restart")
	case "proxy:rebuild":
		commands.Proxy("rebuild")
	case "proxy:prune":
		commands.Proxy("prune")
	case "prune":
		commands.Prune()
	case "rebuild":
		commands.Rebuild()
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
		commands.Uncompress()
	default:
		commands.IsNotDefine()
	}
}

//TODO check opensearchdashboard in browser
//TODO check rabbitMQ in browser
//TODO check redis in browser
