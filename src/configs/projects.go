package configs

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/faradey/madock/src/paths"
)

var generalConfig map[string]string
var projectConfig map[string]string
var projectConfigOnly map[string]string

func GetGeneralConfig() map[string]string {
	if len(generalConfig) == 0 {
		configPath := paths.GetExecDirPath() + "/projects/config.txt"
		if _, err := os.Stat(configPath); !os.IsNotExist(err) && err == nil {
			generalConfig = ParseFile(configPath)
		}

		configPath = paths.GetExecDirPath() + "/config.txt"
		origGeneralConfig := make(map[string]string)
		if _, err := os.Stat(configPath); !os.IsNotExist(err) && err == nil {
			origGeneralConfig = ParseFile(configPath)
		}
		GeneralConfigMapping(origGeneralConfig, generalConfig)
	}

	return generalConfig
}

func GetCurrentProjectConfig() map[string]string {
	return GetProjectConfig(GetProjectName())
}

func GetProjectConfig(projectName string) map[string]string {
	if len(projectConfig) == 0 {
		projectConfig = GetProjectConfigOnly(projectName)
		ConfigMapping(GetGeneralConfig(), projectConfig)
	}

	return projectConfig
}

func GetProjectConfigOnly(projectName string) map[string]string {
	if len(projectConfigOnly) == 0 {
		configPath := paths.GetExecDirPath() + "/projects/" + projectName + "/env.txt"
		if _, err := os.Stat(configPath); os.IsNotExist(err) && err != nil {
			log.Fatal(err)
		}

		projectConfigOnly = ParseFile(configPath)
	}

	return projectConfigOnly
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
