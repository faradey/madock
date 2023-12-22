package versions

import (
	configs2 "github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/paths"
)

func V230() {
	execProjectsDirs := paths.GetDirs(paths.GetExecDirPath() + "/projects")
	execPath := paths.GetExecDirPath() + "/projects/"
	projectName := ""
	envFile := ""
	for _, dir := range execProjectsDirs {
		if paths.IsFileExist(execPath + dir + "/env.txt") {
			projectName = dir
			projectConfOnly := configs2.GetProjectConfigOnly(projectName)
			projectConf := configs2.GetProjectConfig(projectName)
			envFile = paths.MakeDirsByPath(paths.GetExecDirPath()+"/projects/"+projectName) + "/env.txt"
			if _, ok := projectConfOnly["PUBLIC_DIR"]; !ok {
				if projectConf["PLATFORM"] == "magento2" {
					configs2.SetParam(envFile, "PUBLIC_DIR", "pub")
				} else if projectConf["PLATFORM"] == "pwa" {
					configs2.SetParam(envFile, "PUBLIC_DIR", "")
				} else if projectConf["PLATFORM"] == "shopify" {
					configs2.SetParam(envFile, "PUBLIC_DIR", "web/public")
				} else if projectConf["PLATFORM"] == "custom" {
					configs2.SetParam(envFile, "PUBLIC_DIR", "web/public")
				}
			}
		}
	}
}
