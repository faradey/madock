package project

func MakeConfShopware(projectName string) {
	makePhpDockerfile(projectName)
	makeNodeJsDockerfile(projectName)
	makeDBDockerfile(projectName)
	makeElasticDockerfile(projectName)
	makeOpenSearchDockerfile(projectName)
	makeRedisDockerfile(projectName)
	makeKibanaConf(projectName)
	makeScriptsConf(projectName)
}
