package helper

import (
	"fmt"
	"github.com/faradey/madock/src/cli/fmtc"
)

func Help() {
	/* 16 commands */
	fmtc.WarningLn("Usage:")
	tab()
	fmt.Println("command [arguments]")
	fmtc.WarningLn("Available commands:")
	tab()
	fmtc.Success("help")
	tab()
	fmt.Println("Displays help for commands")
	tab()
	fmtc.Success("bash")
	tab()
	fmt.Println("Connect into container using bash [Default container: php]")
	tab()
	tab()
	fmtc.Title("--name")
	tab()
	fmt.Print("Name of container. For example: --name php, --name node, --name db, --name nginx")
	tabln()
	tab()
	tab()
	fmtc.Title("--root")
	tab()
	fmt.Println("Enter to container as root")
	tab()
	fmtc.Success("composer")
	tab()
	fmt.Println("Execute composer inside php container")

	fmt.Println("")
}

func tab() {
	fmt.Print("	")
}

func tabln() {
	fmt.Println("	")
}
