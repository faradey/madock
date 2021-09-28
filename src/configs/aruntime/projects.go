package aruntime

import (
	"github.com/faradey/madock/src/paths"
	"log"
	"os"
)

func CreateProjectConf(projectName string, generalConf map[string]string) {
	paths.MakeDirsByPath(paths.GetExecDirPath() + "/aruntime/projects/" + projectName)
	src := paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/src"
	if _, err := os.Stat(src); os.IsNotExist(err) {
		err := os.Symlink(paths.GetRunDirPath(), src)
		if err != nil {
			log.Fatal(err)
		}
	}

}
