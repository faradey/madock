package configs

import "github.com/faradey/madock/src/helper/paths"

const MainConfigCode = ":config:"
const MadockLevelConfigCode = ":madockconfig:"

func SetParam(projectName, name, value, activeScope, level string) {
	file := ""
	if level != MainConfigCode {
		if level != MadockLevelConfigCode && paths.IsFileExist(paths.GetRunDirPath()+"/.madock/config.xml") {
			file = paths.GetRunDirPath() + "/.madock/config.xml"
		} else {
			if projectName == MadockLevelConfigCode {
				projectName = GetProjectName()
			}

			file = paths.MakeDirsByPath(paths.GetExecDirPath()+"/projects/"+projectName) + "/config.xml"
		}
	} else {
		file = paths.MakeDirsByPath(paths.GetExecDirPath()+"/projects") + "/config.xml"
	}

	confList := ParseXmlFile(file)
	confList = getConfigByScope(confList, activeScope)
	confList[name] = value
	SaveInFile(file, confList, activeScope)
	CleanCache()
}
