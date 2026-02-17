package project

func init() {
	RegisterDockerConfGenerator("shopify", MakeConfShopify)
}

func MakeConfShopify(projectName string) {
	MakePhpDockerfile(projectName)
	MakeDBDockerfile(projectName)
	MakeRedisDockerfile(projectName)
	MakeScriptsConf(projectName)
	MakeClaudeDockerfile(projectName)
}
