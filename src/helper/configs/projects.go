package configs

import (
	"github.com/faradey/madock/src/helper/paths"
	"log"
	"os"
	"strconv"
	"strings"
)

var generalConfig map[string]string
var projectConfig map[string]string
var nameOfProject string

func CleanCache() {
	generalConfig = nil
	projectConfig = nil
	nameOfProject = ""
}

func GetGeneralConfig() map[string]string {
	if len(generalConfig) == 0 {
		generalConfig = GetProjectsGeneralConfig()

		origGeneralConfig := GetOriginalGeneralConfig()
		GeneralConfigMapping(origGeneralConfig, generalConfig)
	}

	return generalConfig
}

func GetOriginalGeneralConfig() map[string]string {
	configPath := paths.GetExecDirPath() + "/config.txt"
	origGeneralConfig := make(map[string]string)
	if _, err := os.Stat(configPath); !os.IsNotExist(err) && err == nil {
		origGeneralConfig = ParseFile(configPath)
	}

	return origGeneralConfig
}

func GetProjectsGeneralConfig() map[string]string {
	generalProjectsConfig := make(map[string]string)
	configPath := paths.GetExecDirPath() + "/projects/config.txt"
	if _, err := os.Stat(configPath); !os.IsNotExist(err) && err == nil {
		generalProjectsConfig = ParseFile(configPath)
	}

	return generalProjectsConfig
}

func GetCurrentProjectConfig() map[string]string {
	if len(projectConfig) == 0 {
		projectConfig = GetProjectConfig(GetProjectName())
	}

	return projectConfig
}

func GetProjectConfig(projectName string) map[string]string {

	config := GetProjectConfigOnly(projectName)
	ConfigMapping(GetGeneralConfig(), config)

	return config
}

func GetProjectConfigOnly(projectName string) map[string]string {
	configPath := paths.GetExecDirPath() + "/projects/" + projectName + "/env.txt"
	if _, err := os.Stat(configPath); os.IsNotExist(err) && err != nil {
		log.Fatal(err)
	}

	return ParseFile(configPath)
}

func GetOption(name string, generalConf, projectConf map[string]string) string {
	if val, ok := projectConf[name]; ok && val != "" {
		return strings.TrimSpace(val)
	}

	if val, ok := generalConf[name]; ok && val != "" {
		return strings.TrimSpace(val)
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
	if nameOfProject == "" {
		for i := 2; i < 1000; i++ {
			nameOfProject = paths.GetRunDirName() + suffix
			envFile = paths.GetExecDirPath() + "/projects/" + nameOfProject + "/env.txt"
			if paths.IsFileExist(envFile) {
				projectConf := GetProjectConfigOnly(nameOfProject)
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
	}

	return nameOfProject
}
