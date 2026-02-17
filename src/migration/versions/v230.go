package versions

import (
	"github.com/faradey/madock/src/helper/paths"
	"github.com/faradey/madock/src/migration/versions/v240/configs"
)

func V230() {
	execProjectsDirs := paths.GetDirs(paths.GetExecDirPath() + "/projects")
	execPath := paths.GetExecDirPath() + "/projects/"
	projectName := ""
	envFile := ""
	for _, dir := range execProjectsDirs {
		if paths.IsFileExist(execPath + dir + "/env.txt") {
			projectName = dir
			projectConfOnly := configs.GetProjectConfigOnly(projectName)
			projectConf := configs.GetProjectConfig(projectName)
			envFile = paths.MakeDirsByPath(paths.GetExecDirPath()+"/projects/"+projectName) + "/env.txt"
			if _, ok := projectConfOnly["PUBLIC_DIR"]; !ok {
				if projectConf["PLATFORM"] == "magento2" {
					configs.SetParam(envFile, "PUBLIC_DIR", "pub")
				} else if projectConf["PLATFORM"] == "shopify" {
					configs.SetParam(envFile, "PUBLIC_DIR", "web/public")
				} else if projectConf["PLATFORM"] == "custom" {
					configs.SetParam(envFile, "PUBLIC_DIR", "public")
				} else if projectConf["PLATFORM"] == "shopware" {
					configs.SetParam(envFile, "PUBLIC_DIR", "public")
				}
			}
		}
	}
}
