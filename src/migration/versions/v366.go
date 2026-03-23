package versions

import (
	"os"

	"github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/paths"
)

// V366 adds php/enabled=true for projects that use PHP but lack the key.
// Older project configs were created before php/enabled existed.
// Without it, the value falls back to the global config (false), which breaks
// docker-compose when xdebug or nginx depends on the php service.
func V366() {
	phpPlatforms := map[string]bool{
		"magento2":   true,
		"shopware":   true,
		"prestashop": true,
	}

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
		migratePhpEnabled(configPath, projectConf, phpPlatforms)

		projectPath := projectConf["path"]
		if projectPath != "" {
			inProjectConfig := projectPath + "/.madock/config.xml"
			if paths.IsFileExist(inProjectConfig) {
				inProjectConf := configs.ParseXmlFile(inProjectConfig)
				inProjectConf = getConfigByScopeV366(inProjectConf, projectConf)
				migratePhpEnabled(inProjectConfig, inProjectConf, phpPlatforms)
			}
		}
	}
}

func migratePhpEnabled(configPath string, projectConf map[string]string, phpPlatforms map[string]bool) {
	if _, has := projectConf["php/enabled"]; has {
		return
	}

	// Check by platform if set
	platform := projectConf["platform"]
	isPHP := phpPlatforms[platform]

	// For old projects without platform key, detect PHP by presence of php/version
	if platform == "" {
		_, hasPhpVersion := projectConf["php/version"]
		isPHP = hasPhpVersion
	}

	if !isPHP {
		return
	}

	config := new(configs.ConfigLines)
	config.EnvFile = configPath
	config.ActiveScope = "default"
	if scope, ok := projectConf["activeScope"]; ok {
		config.ActiveScope = scope
	}

	config.Set("php/enabled", "true")
	config.Save()
}

func getConfigByScopeV366(rawConf, fallbackConf map[string]string) map[string]string {
	result := make(map[string]string)
	activeScope := "default"
	if scope, ok := rawConf["activeScope"]; ok {
		activeScope = scope
	}
	prefix := "scopes/" + activeScope + "/"
	for key, val := range rawConf {
		if len(key) > len(prefix) && key[:len(prefix)] == prefix {
			result[key[len(prefix):]] = val
		}
	}
	if scope, ok := fallbackConf["activeScope"]; ok {
		result["activeScope"] = scope
	}
	return result
}
