package project

func MakeConfPWA(projectName string) {
	makeNodeJsDockerfile(projectName)
	makeClaudeDockerfile(projectName)
}
