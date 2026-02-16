package versions

import (
	"os"
	"strings"

	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/paths"
)

// V320 adds the "language" field to existing project configs for backward compatibility
func V320() {
	projectsDir := paths.GetExecDirPath() + "/projects"
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		projectName := entry.Name()
		configPath := projectsDir + "/" + projectName + "/config.xml"
		if !paths.IsFileExist(configPath) {
			continue
		}

		projectConf := configs.GetProjectConfigOnly(projectName)
		if _, ok := projectConf["language"]; ok && projectConf["language"] != "" {
			continue // Already has language set
		}

		// Determine language from platform
		platform := projectConf["platform"]
		language := "php"
		switch platform {
		case "pwa":
			language = "nodejs"
		case "magento2", "shopware", "prestashop", "shopify", "custom":
			language = "php"
		}

		// Also check in-project config
		projectPath := projectConf["path"]
		if projectPath != "" {
			inProjectConfig := projectPath + "/.madock/config.xml"
			if paths.IsFileExist(inProjectConfig) {
				config := new(configs.ConfigLines)
				config.EnvFile = inProjectConfig
				config.ActiveScope = "default"
				if scope, ok := projectConf["activeScope"]; ok {
					config.ActiveScope = scope
				}
				config.Set("language", language)
				config.Save()
				continue
			}
		}

		config := new(configs.ConfigLines)
		config.EnvFile = configPath
		config.ActiveScope = "default"
		if scope, ok := projectConf["activeScope"]; ok {
			config.ActiveScope = scope
		}
		config.Set("language", language)
		config.Save()
	}

	// Also check current directory for .madock/config.xml
	currentPath := paths.GetRunDirPath()
	inProjectConfig := currentPath + "/.madock/config.xml"
	if paths.IsFileExist(inProjectConfig) {
		rawConf := configs.ParseXmlFile(inProjectConfig)
		if _, ok := rawConf["language"]; !ok || rawConf["language"] == "" {
			platform := rawConf["platform"]
			language := "php"
			if strings.TrimSpace(platform) == "pwa" {
				language = "nodejs"
			}
			config := new(configs.ConfigLines)
			config.EnvFile = inProjectConfig
			config.ActiveScope = "default"
			config.Set("language", language)
			config.Save()
		}
	}
}
