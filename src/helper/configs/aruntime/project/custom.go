package project

func MakeConfCustom(projectName string) {
	makePhpDockerfile(projectName)
	makeDBDockerfile(projectName)
	makeElasticDockerfile(projectName)
	makeOpenSearchDockerfile(projectName)
	makeRedisDockerfile(projectName)
	makeKibanaConf(projectName)
	makeScriptsConf(projectName)
	processOtherCTXFiles(projectName)
}
