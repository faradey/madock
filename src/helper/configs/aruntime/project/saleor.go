package project

func init() {
	RegisterDockerConfGenerator("saleor", MakeConfSaleor)
}

func MakeConfSaleor(projectName string) {
	MakeDockerfile(projectName, "python/Dockerfile", "python.Dockerfile")
	MakeDBDockerfile(projectName)
	MakeRedisDockerfile(projectName)
	MakeScriptsConf(projectName)
	MakeClaudeDockerfile(projectName)
}
