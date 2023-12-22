package versions

import (
	configs2 "github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/paths"
)

func V220() {
	execProjectsDirs := paths.GetDirs(paths.GetExecDirPath() + "/projects")
	execPath := paths.GetExecDirPath() + "/projects/"
	projectName := ""
	envFile := ""
	for _, dir := range execProjectsDirs {
		if paths.IsFileExist(execPath + dir + "/env.txt") {
			projectName = dir
			projectConf := configs2.GetProjectConfigOnly(projectName)
			envFile = paths.MakeDirsByPath(paths.GetExecDirPath()+"/projects/"+projectName) + "/env.txt"
			if _, ok := projectConf["UBUNTU_VERSION"]; !ok {
				configs2.SetParam(envFile, "UBUNTU_VERSION", "20.04")
			}
			if _, ok := projectConf["CONTAINER_NAME_PREFIX"]; !ok {
				configs2.SetParam(envFile, "CONTAINER_NAME_PREFIX", "")
			}
		}
	}
}
