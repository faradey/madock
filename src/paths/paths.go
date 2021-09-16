package paths

import (
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

func GetRunDirPath() string {
	return filepath.Dir(os.Args[0])
}

func GetRunDirName() string {
	dir, err := filepath.Abs(GetRunDirPath())
	if err != nil {
		return ""
	}

	return dir
}
