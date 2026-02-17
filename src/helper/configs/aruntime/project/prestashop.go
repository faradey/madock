package project

func init() {
	RegisterDockerConfGenerator("prestashop", MakeConfPrestashop)
}

func MakeConfPrestashop(projectName string) {
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
