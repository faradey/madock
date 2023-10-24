package project

func MakeConfMagento2(projectName string) {
	makePhpDockerfile(projectName)
	makeDBDockerfile(projectName)
	makeElasticDockerfile(projectName)
	makeOpenSearchDockerfile(projectName)
	makeRedisDockerfile(projectName)
	makeKibanaConf(projectName)
	makeScriptsConf(projectName)
	processOtherCTXFiles(projectName)
}
