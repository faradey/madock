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
		case "refresh":
		case "rebuild":
		case "magento":
		case "composer":
		case "dbimport":
		case "dbexport":
		case "help":
			helper.Help()
		default:
			commands.IsNotDefine()
		}
	} else {
		helper.Help()
	}
}
