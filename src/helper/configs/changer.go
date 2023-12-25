package configs

func SetParam(file, name, value, activeScope string) {
	confList := ParseXmlFile(file)
	confList = getConfigByScope(confList, activeScope)
	confList[name] = value
	SaveInFile(file, confList, activeScope)
	CleanCache()
}
