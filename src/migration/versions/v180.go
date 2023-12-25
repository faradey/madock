package versions

import (
	"github.com/faradey/madock/src/helper/paths"
	"github.com/faradey/madock/src/migration/versions/v240/configs"
	"os"
)

func V180() {
	execProjectsDirs := paths.GetDirs(paths.GetExecDirPath() + "/projects")
	execPath := paths.GetExecDirPath() + "/projects/"
	projectName := ""
	for _, dir := range execProjectsDirs {
		if paths.IsFileExist(execPath + dir + "/env.txt") {
			projectName = dir
			projectConf := configs.GetProjectConfig(projectName)
			if _, ok := projectConf["PATH"]; !ok {
				if fi, err := os.Lstat(paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/src"); err == nil {
					if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
						link, err := os.Readlink(paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/src")
						if err == nil {
							configs.SetParam(execPath+dir+"/env.txt", "PATH", link)
						}
					}
				}
			}
		}
	}
}
