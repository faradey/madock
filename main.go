package main

import (
	"os"
	"strings"

	"github.com/faradey/madock/src/cli/attr"
	"github.com/faradey/madock/src/cli/commands"
	"github.com/faradey/madock/src/cli/helper"
	"github.com/faradey/madock/src/migration"
)

var appVersion string = "1.7.0"

func main() {
	if len(os.Args) > 1 {
		migration.Apply(appVersion)
		command := strings.ToLower(os.Args[1])
		attr.ParseAttributes()

		switch command {
		case "bash":
			commands.Bash()
		case "c:f":
			commands.CleanCache()
		case "magento-cloud", "cloud":
			commands.Cloud(strings.Join(os.Args[2:], " "))
		case "cli":
			commands.Cli(strings.Join(os.Args[2:], " "))
		case "composer":
			commands.Composer(strings.Join(os.Args[2:], " "))
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
		case "info":
			commands.Info()
		case "help":
			helper.Help()
		case "logs":
			commands.Logs()
		case "magento", "m":
			commands.Magento(strings.Join(os.Args[2:], " "))
		case "node":
			commands.Node(strings.Join(os.Args[2:], " "))
		case "patch:create":
			commands.PatchCreate()
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
	} else {
		helper.Help()
	}
}
