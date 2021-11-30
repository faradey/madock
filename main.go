package main

import (
	"github.com/faradey/madock/src/cli/commands"
	"github.com/faradey/madock/src/cli/fmtc"
	"github.com/faradey/madock/src/cli/helper"
	"os"
	"strings"
)

func main() {
	if len(os.Args) > 1 {
		command := strings.ToLower(os.Args[1])
		flag := ""
		if len(os.Args) > 2 {
			flag = strings.ToLower(os.Args[2])
		}

		switch command {
		case "bash":
			commands.Bash(flag)
		case "composer":
			flag = strings.Join(os.Args[2:], " ")
			commands.Composer(flag)
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
		case "help":
			helper.Help()
		case "logs":
			commands.Logs(flag)
		case "magento":
			flag = strings.Join(os.Args[2:], " ")
			commands.Magento(flag)
		case "node":
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
		case "setup":
			commands.Setup()
		case "start":
			commands.Start()
		case "stop":
			commands.Stop()
		default:
			commands.IsNotDefine()
		}
	} else {
		helper.Help()
	}
}
