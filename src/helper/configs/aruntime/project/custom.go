package project

import "github.com/faradey/madock/src/helper/configs"

func MakeConfCustom(projectName string) {
	projectConf := configs.GetProjectConfig(projectName)
	language := projectConf["language"]
	if language == "" {
		language = "php"
	}

	makeMainContainerDockerfile(projectName)

	if language == "php" {
		makeNodeJsDockerfile(projectName)
	}

	makeDBDockerfile(projectName)
	makeElasticDockerfile(projectName)
	makeOpenSearchDockerfile(projectName)
	makeRedisDockerfile(projectName)
	makeKibanaConf(projectName)
	makeScriptsConf(projectName)
	makeClaudeDockerfile(projectName)
}
