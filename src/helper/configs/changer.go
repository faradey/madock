package configs

func SetParam(file, name, value string) {
	confList := ParseFile(file)

	confList[name] = value

	SaveInFile(file, confList)
	CleanCache()
}
