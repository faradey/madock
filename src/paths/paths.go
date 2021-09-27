package paths

import (
	"log"
	"os"
	"path/filepath"
)

func GetExecDirPath() string {
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

func GetExecDirName() string {
	return filepath.Base(GetExecDirPath())
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
