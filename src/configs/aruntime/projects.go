package aruntime

import (
	"github.com/faradey/madock/src/paths"
	"log"
	"os"
)

func CreateProjectConf(projectName string) {
	paths.MakeDirsByPath(paths.GetExecDirPath() + "/aruntime/projects/" + projectName)
	src := paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/src"
	if _, err := os.Lstat(src); err == nil {
		if err := os.Remove(src); err != nil {
			log.Fatalf("failed to unlink: %+v", err)
		}
	}

	err := os.Symlink(paths.GetRunDirPath(), src)
	if err != nil {
		log.Fatal(err)
	}

}
