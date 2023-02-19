package paths

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/faradey/madock/src/functions"
)

func GetExecDirPath() string {
	var dirAbsPath string

	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exReal, err := filepath.EvalSymlinks(ex)
	if err != nil {
		dirAbsPath = filepath.Dir(ex)
		return dirAbsPath
	} else {
		dirAbsPath = filepath.Dir(exReal)
		return dirAbsPath
	}

	panic("Unknown error")
}

func GetExecDirName() string {
	return filepath.Base(GetExecDirPath())
}

func GetExecDirNameByPath(path string) string {
	return filepath.Base(path)
}

func GetRunDirPath() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return ""
	}

	return dir
}

func GetRunDirName() string {
	return filepath.Base(GetRunDirPath())
}

func GetRunDirNameWithHash() string {
	return filepath.Base(GetRunDirPath()) + "__" + strconv.Itoa(int(functions.Hash(GetRunDirPath())))
}

func GetDirs(path string) (dirs []string) {
	items, err := os.ReadDir(path)
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range items {
		if file.IsDir() {
			dirs = append(dirs, file.Name())
		}
	}

	return dirs
}

func GetFiles(path string) (dirs []string) {
	items, err := os.ReadDir(path)
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range items {
		if !file.IsDir() {
			dirs = append(dirs, file.Name())
		}
	}

	return dirs
}

func GetFilesRecursively(path string) (dirs []string) {
	items, err := os.ReadDir(path)
	if err == nil {
		for _, file := range items {
			if !file.IsDir() {
				dirs = append(dirs, path+"/"+file.Name())
			} else {
				dirs = append(dirs, GetFilesRecursively(path+"/"+file.Name())...)
			}
		}
	}

	return dirs
}

func GetDBFiles(path string) (dirs []string) {
	items, err := os.ReadDir(path)
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range items {
		if !file.IsDir() {
			if file.Name()[0:1] != "." && strings.Contains(strings.ToLower(file.Name()), ".sql") && !strings.Contains(strings.ToLower(path), "/dev/tests/acceptance") {
				dirs = append(dirs, path+"/"+file.Name())
			}
		} else {
			dirs = append(dirs, GetDBFiles(path+"/"+file.Name())...)
		}
	}

	return dirs
}
