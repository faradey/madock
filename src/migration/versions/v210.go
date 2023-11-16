package versions

import (
	"os"

	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/paths"
)

func V210() {
	execProjectsDirs := paths.GetDirs(paths.GetExecDirPath() + "/projects")
	execPath := paths.GetExecDirPath() + "/projects/"
	projectName := ""
	envFile := ""
	for _, dir := range execProjectsDirs {
		if _, err := os.Stat(execPath + dir + "/env.txt"); !os.IsNotExist(err) {
			projectName = dir
			projectConf := configs.GetProjectConfigOnly(projectName)
			if _, ok := projectConf["UBUNTU_VERSION"]; !ok {
				envFile = paths.MakeDirsByPath(paths.GetExecDirPath()+"/projects/"+projectName) + "/env.txt"
				configs.SetParam(envFile, "UBUNTU_VERSION", "20.04")
			}
		}
	}
}
