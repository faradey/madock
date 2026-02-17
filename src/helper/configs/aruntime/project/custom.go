package project

import "github.com/faradey/madock/v3/src/helper/configs"

func init() {
	RegisterDockerConfGenerator("custom", MakeConfCustom)
}

func MakeConfCustom(projectName string) {
	projectConf := configs.GetProjectConfig(projectName)
	language := projectConf["language"]
	if language == "" {
		language = "php"
	}

	MakeMainContainerDockerfile(projectName)

	if language == "php" {
		MakeNodeJsDockerfile(projectName)
	}

	MakeDBDockerfile(projectName)
	MakeElasticDockerfile(projectName)
	MakeOpenSearchDockerfile(projectName)
	MakeRedisDockerfile(projectName)
	MakeKibanaConf(projectName)
	MakeScriptsConf(projectName)
	MakeClaudeDockerfile(projectName)
}
