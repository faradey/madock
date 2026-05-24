package project

func init() {
	RegisterDockerConfGenerator("spree", MakeConfSpree)
}

func MakeConfSpree(projectName string) {
	MakeDockerfile(projectName, "ruby/Dockerfile", "ruby.Dockerfile")
	MakeDockerfile(projectName, "storefront/Dockerfile", "storefront.Dockerfile")
	MakeDBDockerfile(projectName)
	MakeRedisDockerfile(projectName)
	MakeScriptsConf(projectName)
	MakeClaudeDockerfile(projectName)
}
