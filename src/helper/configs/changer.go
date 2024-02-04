package configs

import "github.com/faradey/madock/src/helper/paths"

const MainConfigCode = ":config:"

func SetParam(projectName, name, value, activeScope string) {
	file := ""
	if projectName != MainConfigCode {
		if paths.GetRunDirName() == projectName && paths.IsFileExist(paths.GetRunDirPath()+"/.madock/config.xml") {
			file = paths.GetRunDirPath() + "/.madock/config.xml"
		} else {
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
