package project

func init() {
	RegisterDockerConfGenerator("magento2", MakeConfMagento2)
}

func MakeConfMagento2(projectName string) {
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
