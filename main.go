package main

import (
	"fmt"
	"github.com/faradey/madock/src/paths"
)

func main() {
	fmt.Println(paths.GetExecDirPath())
	fmt.Println(paths.GetRunDirName())
}
