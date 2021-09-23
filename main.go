package main

import (
	"fmt"
	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/paths"
	"os"
)

func main() {
	if len(os.Args) > 0 {
		command := os.Args[1]
		fmt.Println(command)
		fmt.Println(paths.GetExecDirName())
		fmt.Println(paths.GetExecDirPath())
		fmt.Println(paths.GetRunDirPath())
		fmt.Println(paths.GetRunDirName())
		fmt.Println(configs.GetProjectsConfig(paths.GetExecDirPath() + "/projects"))
	}
}
