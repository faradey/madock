package project

func init() {
	RegisterDockerConfGenerator("woocommerce", MakeConfWoocommerce)
}

func MakeConfWoocommerce(projectName string) {
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
