package configs

import (
	"github.com/faradey/madock/src/helper/logger"
	"github.com/faradey/madock/src/helper/paths"
	"os"
)

var generalConfig map[string]string

func CleanCache() {
	generalConfig = nil
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

func GetProjectConfig(projectName string) map[string]string {

	config := GetProjectConfigOnly(projectName)
	ConfigMapping(GetGeneralConfig(), config)

	return config
}

func GetProjectConfigOnly(projectName string) map[string]string {
	configPath := paths.GetExecDirPath() + "/projects/" + projectName + "/env.txt"
	if _, err := os.Stat(configPath); os.IsNotExist(err) && err != nil {
		logger.Fatal(err)
	}

	return ParseFile(configPath)
}
