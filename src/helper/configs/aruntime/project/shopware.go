package project

func init() {
	RegisterDockerConfGenerator("shopware", MakeConfShopware)
}

func MakeConfShopware(projectName string) {
	MakePhpDockerfile(projectName)
	MakeNodeJsDockerfile(projectName)
	MakeDBDockerfile(projectName)
	MakeElasticDockerfile(projectName)
	MakeOpenSearchDockerfile(projectName)
	MakeRedisDockerfile(projectName)
	MakeKibanaConf(projectName)
	MakeScriptsConf(projectName)
	MakeClaudeDockerfile(projectName)
}
