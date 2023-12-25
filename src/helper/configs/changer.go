package configs

func SetParam(file, name, value string) {
	confList := ParseXmlFile(file)

	confList[name] = value

	SaveInFile(file, confList)
	CleanCache()
}
