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

	// The nodejs service is rendered into docker-compose.yml whenever
	// nodejs/enabled is set, whatever the language — but its Dockerfile was
	// only written for php projects. A Python, Go, Ruby or language-less
	// project with Node enabled got a compose service pointing at a file
	// nobody generated: missing on a fresh project, and worse on an old one,
	// where a copy left over from an earlier madock is silently built instead.
	//
	// Not when the language is nodejs. There the node container *is* the main
	// service and MakeMainContainerDockerfile has already written this same
	// file from the language template, which the service template does not
	// match — it has no cron. Writing it again here would take that away.
	if language == "php" || (projectConf["nodejs/enabled"] == "true" && language != "nodejs") {
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
