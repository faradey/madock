package configs

import (
	"bytes"
	"encoding/xml"
	"github.com/faradey/madock/src/helper/paths"
	"github.com/go-xmlfmt/xmlfmt"
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
	configPath := paths.GetExecDirPath() + "/config.xml"
	origGeneralConfig := make(map[string]string)
	if _, err := os.Stat(configPath); !os.IsNotExist(err) && err == nil {
		origGeneralConfig = ParseXmlFile(configPath)
		origGeneralConfig = getConfigByScope(origGeneralConfig, "default")
	}
	return origGeneralConfig
}

func GetProjectsGeneralConfig() map[string]string {
	generalProjectsConfig := make(map[string]string)
	configPath := paths.GetExecDirPath() + "/projects/config.xml"
	if _, err := os.Stat(configPath); !os.IsNotExist(err) && err == nil {
		generalProjectsConfig = ParseXmlFile(configPath)
		generalProjectsConfig = getConfigByScope(generalProjectsConfig, "default")
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
	activeConfig := make(map[string]string)
	configPath := paths.GetExecDirPath() + "/projects/" + projectName + "/config.xml"
	activeScope := "default"
	if paths.IsFileExist(configPath) {
		config := ParseXmlFile(configPath)

		defaultConfig := getConfigByScope(config, activeScope)
		if v, ok := config["activeScope"]; ok {
			activeScope = v
			activeConfig = getConfigByScope(config, activeScope)
		}

		ConfigMapping(defaultConfig, activeConfig)
		activeConfig["activeScope"] = activeScope
	}

	defaultConfig := GetProjectConfigInProject(activeConfig["path"])
	activeProjectConfig := make(map[string]string)
	ConfigMapping(defaultConfig, activeProjectConfig)
	ConfigMapping(activeConfig, activeProjectConfig)
	return activeProjectConfig
}

func GetCurrentProjectConfigPath(projectName string) string {
	if projectName == "" {
		projectName = GetProjectName()
	}
	envFile := paths.MakeDirsByPath(paths.GetExecDirPath()+"/projects/"+projectName) + "/config.xml"
	if paths.IsFileExist(paths.GetRunDirPath() + "/.madock/config.xml") {
		envFile = paths.GetRunDirPath() + "/.madock/config.xml"
	}

	return envFile
}

func GetProjectConfigInProject(projectPath string) map[string]string {
	configPath := projectPath + "/.madock/config.xml"
	if !paths.IsFileExist(configPath) {
		return make(map[string]string)
	}

	config := ParseXmlFile(configPath)
	activeConfig := make(map[string]string)
	activeScope := "default"
	defaultConfig := getConfigByScope(config, activeScope)
	if v, ok := config["activeScope"]; ok {
		activeScope = v
		activeConfig = getConfigByScope(config, activeScope)
	}

	ConfigMapping(defaultConfig, activeConfig)
	activeConfig["activeScope"] = activeScope
	return activeConfig
}

func GetOption(name string, generalConf, projectConf map[string]string) string {
	if val, ok := projectConf[name]; ok && val != "" {
		return strings.TrimSpace(val)
	} else if val, ok := generalConf[name]; ok && val != "" {
		return strings.TrimSpace(val)
	}

	return ""
}

func PrepareDirsForProject(projectName string) {
	projectPath := paths.GetExecDirPath() + "/projects/" + projectName
	paths.MakeDirsByPath(projectPath)
	paths.MakeDirsByPath(projectPath + "/docker")
	paths.MakeDirsByPath(projectPath + "/docker/nginx")
	paths.MakeDirsByPath(projectPath + "/docker/php")
}

func GetProjectName() string {
	suffix := ""
	if nameOfProject == "" {
		for i := 2; i < 1000; i++ {
			nameOfProject = paths.GetRunDirName() + suffix
			if paths.IsFileExist(paths.GetExecDirPath() + "/projects/" + nameOfProject + "/config.xml") {
				projectConf := GetProjectConfigOnly(nameOfProject)
				val, ok := projectConf["path"]
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

func getConfigByScope(originConfig map[string]string, activeScope string) map[string]string {
	config := make(map[string]string)
	for key, val := range originConfig {
		if strings.Index(key, "scopes/"+activeScope+"/") == 0 {
			config[key[len("scopes/"+activeScope+"/"):]] = val
		}
		if key == "scopes/activeScope" {
			config[key] = val
		}
	}

	return config
}

func GetScopes(projectName string) map[string]string {
	scopes := make(map[string]string)
	configPath := GetCurrentProjectConfigPath(projectName)
	if _, err := os.Stat(configPath); os.IsNotExist(err) && err != nil {
		return scopes
	}

	config := ParseXmlFile(configPath)

	var parts []string
	for key, _ := range config {
		parts = strings.Split(key, "/")
		if len(parts) > 1 && parts[0] == "scopes" {
			if val, ok := config["activeScope"]; !ok || val == parts[1] {
				scopes[parts[1]] = "1"
				continue
			}

			scopes[parts[1]] = "0"
		}
	}

	return scopes
}

func SetScope(projectName, scope string) bool {
	configPath := GetCurrentProjectConfigPath(projectName)
	if _, err := os.Stat(configPath); os.IsNotExist(err) && err != nil {
		return false
	}

	config := ParseXmlFile(configPath)
	config["activeScope"] = scope
	resultData := make(map[string]interface{})
	for key, value := range config {
		resultData[key] = value
	}
	resultMapData := SetXmlMap(resultData)
	w := &bytes.Buffer{}
	w.WriteString(xml.Header)
	err := MarshalXML(resultMapData, xml.NewEncoder(w), "config")
	if err != nil {
		log.Fatalln(err)
	}
	err = os.WriteFile(configPath, []byte(xmlfmt.FormatXML(w.String(), "", "    ", true)), 0755)
	if err != nil {
		return false
	}

	return true
}

func AddScope(projectName, scope string) bool {
	configPath := GetCurrentProjectConfigPath(projectName)
	if _, err := os.Stat(configPath); os.IsNotExist(err) && err != nil {
		return false
	}

	config := ParseXmlFile(configPath)
	config["activeScope"] = scope
	config[scope] = ""
	resultData := make(map[string]interface{})
	for key, value := range config {
		resultData[key] = value
	}
	resultMapData := SetXmlMap(resultData)
	w := &bytes.Buffer{}
	w.WriteString(xml.Header)
	err := MarshalXML(resultMapData, xml.NewEncoder(w), "config")
	if err != nil {
		log.Fatalln(err)
	}
	err = os.WriteFile(configPath, []byte(xmlfmt.FormatXML(w.String(), "", "    ", true)), 0755)
	if err != nil {
		return false
	}

	return true
}

func GetActiveScope(projectName string, withDefault bool, prefix string) string {
	config := GetProjectConfig(projectName)
	if val, ok := config["activeScope"]; ok && val != "default" {
		return prefix + val
	}

	if withDefault {
		return prefix + "default"
	}

	return ""
}
