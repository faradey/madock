package main

import (
	"os"
	"strings"

	"github.com/faradey/madock/src/cli/attr"
	"github.com/faradey/madock/src/cli/commands"
	"github.com/faradey/madock/src/cli/fmtc"
	"github.com/faradey/madock/src/cli/helper"
	"github.com/faradey/madock/src/migration"
)

var appVersion string = "1.6.0"

func main() {
	if len(os.Args) > 1 {
		migration.Apply(appVersion)
		command := strings.ToLower(os.Args[1])
		flag := ""
		if len(os.Args) > 2 {
			flag = strings.ToLower(os.Args[2])
		}
		attr.ParseAttributes()

		switch command {
		case "bash":
			commands.Bash(flag)
		case "c:f":
			commands.CleanCache()
		case "magento-cloud", "cloud":
			flag = strings.Join(os.Args[2:], " ")
			commands.Cloud(flag)
		case "composer":
			flag = strings.Join(os.Args[2:], " ")
			commands.Composer(flag)
		case "compress":
			commands.Compress()
		case "config":
			optionName := ""
			if len(os.Args) > 3 {
				optionName = strings.ToLower(os.Args[3])
			}
			var flags []string
			if len(os.Args) > 4 {
				flags = os.Args[4:]
			}
			if flag == "set" {
				commands.SetEnvOption(optionName, flags)
			} else if flag == "show" {
				commands.ShowEnv()
			} else {
				fmtc.ErrorLn("The command is not defined. Run 'madock help' to invoke help")
			}
		case "cron":
			commands.Cron(flag)
		case "db":
			option := ""
			if len(os.Args) > 3 {
				option = strings.ToLower(os.Args[3])
			}
			commands.DB(flag, option)
		case "debug":
			commands.Debug(flag)
		case "info":
			commands.Info()
		case "help":
			helper.Help()
		case "logs":
			commands.Logs(flag)
		case "magento", "m":
			flag = strings.Join(os.Args[2:], " ")
			commands.Magento(flag)
		case "node":
			flag = strings.Join(os.Args[2:], " ")
			commands.Node(flag)
		case "proxy":
			commands.Proxy(flag)
		case "prune":
			commands.Prune()
		case "rebuild":
			commands.Rebuild()
		case "remote":
			option := ""
			if len(os.Args) > 3 {
				option = strings.ToLower(os.Args[3])
			}
			commands.Remote(flag, option)
		case "restart":
			commands.Restart()
		case "service":
			option := ""
			if len(os.Args) > 3 {
				option = strings.ToLower(os.Args[3])
			}
			commands.SwitchService(flag, option)
		case "setup":
			commands.Setup()
		case "ssl":
			commands.Ssl(flag)
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
