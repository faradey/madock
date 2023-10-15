package configs

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/faradey/madock/src/paths"
)

func GetGeneralConfig() map[string]string {
	configPath := paths.GetExecDirPath() + "/projects/config.txt"
	generalConfig := make(map[string]string)
	if _, err := os.Stat(configPath); !os.IsNotExist(err) && err == nil {
		generalConfig = ParseFile(configPath)
	}

	configPath = paths.GetExecDirPath() + "/config.txt"
	origGeneralConfig := make(map[string]string)
	if _, err := os.Stat(configPath); !os.IsNotExist(err) && err == nil {
		origGeneralConfig = ParseFile(configPath)
	}
	GeneralConfigMapping(origGeneralConfig, generalConfig)

	return generalConfig
}

func GetCurrentProjectConfig() map[string]string {
	return GetProjectConfig(GetProjectName())
}

func GetProjectConfig(projectName string) map[string]string {
	configPath := paths.GetExecDirPath() + "/projects/" + projectName + "/env.txt"
	if _, err := os.Stat(configPath); os.IsNotExist(err) && err != nil {
		log.Fatal(err)
	}

	config := ParseFile(configPath)
	ConfigMapping(GetGeneralConfig(), config)

	return config
}

func GetOption(name string, generalConfig, projectConfig map[string]string) string {
	if val, ok := projectConfig[name]; ok && val != "" {
		return strings.TrimSpace(val)
	}

	if val, ok := generalConfig[name]; ok && val != "" {
		return strings.TrimSpace(generalConfig[name])
	}
	return ""
}

func PrepareDirsForProject(projectName string) {
	projectPath := paths.GetExecDirPath() + "/projects/" + projectName
	paths.MakeDirsByPath(projectPath)
	paths.MakeDirsByPath(projectPath + "/docker")
	paths.MakeDirsByPath(projectPath + "/docker/nginx")
}

func GetProjectName() string {
	suffix := ""
	envFile := ""
	name := ""

	for i := 2; i < 1000; i++ {
		name = paths.GetRunDirName() + suffix
		envFile = paths.GetExecDirPath() + "/projects/" + name + "/env.txt"
		if _, err := os.Stat(envFile); !os.IsNotExist(err) {
			projectConf := GetProjectConfig(name)
			val, ok := projectConf["PATH"]
			if ok && val != paths.GetRunDirPath() {
				suffix = "-" + strconv.Itoa(i)
			} else {
				break
			}
		} else {
			break
		}
	}

	return name
}
