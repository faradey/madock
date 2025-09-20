package project

func MakeConfShopify(projectName string) {
	makePhpDockerfile(projectName)
	makeDBDockerfile(projectName)
	makeRedisDockerfile(projectName)
	makeScriptsConf(projectName)
	makeClaudeDockerfile(projectName)
}
