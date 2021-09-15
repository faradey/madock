package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	fmt.Println(getDir())
}

func getDir() string {
	var dirAbsPath string
	ex, err := os.Executable()
	if err == nil {
		dirAbsPath = filepath.Dir(ex)
		return dirAbsPath
	}

	exReal, err := filepath.EvalSymlinks(ex)
	if err != nil {
		panic(err)
	}
	dirAbsPath = filepath.Dir(exReal)
	return dirAbsPath
}
