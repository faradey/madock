package project

func init() {
	RegisterDockerConfGenerator("sylius", MakeConfSylius)
}

func MakeConfSylius(projectName string) {
	MakePhpDockerfile(projectName)
	MakeDBDockerfile(projectName)
	MakeElasticDockerfile(projectName)
	MakeOpenSearchDockerfile(projectName)
	MakeRedisDockerfile(projectName)
	MakeScriptsConf(projectName)
	MakeClaudeDockerfile(projectName)
}
