package configs

import "github.com/faradey/madock/v3/src/helper/paths"

const MainConfigCode = ":config:"
const MadockLevelConfigCode = ":madockconfig:"

func SetParam(projectName, name, value, activeScope, level string) {
	file := ""
	if level == MainConfigCode {
		file = paths.MakeDirsByPath(paths.GetExecDirPath()+"/projects") + "/config.xml"
	} else {
		if projectName == MadockLevelConfigCode {
			projectName = GetProjectName()
		}

		file = paths.MakeDirsByPath(paths.GetExecDirPath()+"/projects/"+projectName) + "/config.xml"
	}
	confList := make(map[string]string)
	if paths.IsFileExist(file) {
		confList = ParseXmlFile(file)
	}
	confList = getConfigByScope(confList, activeScope)
	confList[name] = value
	SaveInFile(file, confList, activeScope)
	CleanCache()
}
