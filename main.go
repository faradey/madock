package main

import (
	"fmt"
	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/paths"
)

func main() {
	fmt.Println(paths.GetExecDirName())
	fmt.Println(paths.GetExecDirPath())
	fmt.Println(paths.GetRunDirPath())
	fmt.Println(paths.GetRunDirName())
	fmt.Println(configs.GetProjectsConfig(paths.GetExecDirPath() + "/projects"))
}
