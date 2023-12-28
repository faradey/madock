package configs

import "github.com/faradey/madock/src/helper/paths"

const MainConfigCode = ":config:"

func SetParam(projectName, name, value, activeScope string) {
	file := ""
	if projectName != MainConfigCode {
		file = paths.MakeDirsByPath(paths.GetExecDirPath()+"/projects/"+projectName) + "/config.xml"
	} else {
		file = paths.MakeDirsByPath(paths.GetExecDirPath()+"/projects") + "/config.xml"
	}

	confList := ParseXmlFile(file)
	confList = getConfigByScope(confList, activeScope)
	confList[name] = value
	SaveInFile(file, confList, activeScope)
	CleanCache()
}
