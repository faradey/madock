package main

import (
	"github.com/faradey/madock/src/cli/commands"
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
		case "setup":
			commands.Setup()
		case "start":
			commands.Start()
		case "stop":
			commands.Stop(flag)
		case "restart":
			commands.Start()
		case "refresh":
			commands.Start()
		case "rebuild":
			commands.Start()
		case "magento":
			flag = strings.Join(os.Args[2:], " ")
			commands.Magento(flag)
		case "composer":
			flag = strings.Join(os.Args[2:], " ")
			commands.Composer(flag)
		case "db --import":
		case "db --export":
		case "cron":
		case "bash":
		case "help":
			helper.Help()
		default:
			commands.IsNotDefine()
		}
	} else {
		helper.Help()
	}
}
