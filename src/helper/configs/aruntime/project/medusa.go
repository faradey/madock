package project

func init() {
	RegisterDockerConfGenerator("medusa", MakeConfMedusa)
}

func MakeConfMedusa(projectName string) {
	MakeDockerfile(projectName, "Dockerfile", "nodejs.Dockerfile")
	MakeDockerfile(projectName, "storefront/Dockerfile", "storefront.Dockerfile")
	MakeDBDockerfile(projectName)
	MakeRedisDockerfile(projectName)
	MakeScriptsConf(projectName)
	MakeClaudeDockerfile(projectName)
}
