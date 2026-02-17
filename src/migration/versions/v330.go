package versions

import (
	"os"

	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/paths"
)

// V330 migrates config for the language unification:
// 1. Renames php/timezone -> timezone
// 2. Adds php/enabled=true for PHP-based projects
// 3. Renames nodejs/enabled -> php/nodejs/enabled for PHP projects on custom platform
func V330() {
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

		// Migrate PWA platform to custom+nodejs
		migratePWAToCustom(configPath, projectConf)

		// Also check in-project config for PWA migration
		projectPath := projectConf["path"]
		if projectPath != "" {
			inProjectConfig := projectPath + "/.madock/config.xml"
			if paths.IsFileExist(inProjectConfig) {
				inProjectConf := configs.ParseXmlFile(inProjectConfig)
				migratePWAToCustom(inProjectConfig, inProjectConf)
			}
		}

		// Re-read config after potential PWA migration
		projectConf = configs.GetProjectConfigOnly(projectName)
		migrateProjectConfig(configPath, projectConf)

		// Check in-project config for other migrations
		if projectPath != "" {
			inProjectConfig := projectPath + "/.madock/config.xml"
			if paths.IsFileExist(inProjectConfig) {
				inProjectConf := configs.ParseXmlFile(inProjectConfig)
				migrateInProjectConfig(inProjectConfig, inProjectConf, projectConf)
			}
		}
	}

	// Also check current directory for .madock/config.xml
	currentPath := paths.GetRunDirPath()
	inProjectConfig := currentPath + "/.madock/config.xml"
	if paths.IsFileExist(inProjectConfig) {
		rawConf := configs.ParseXmlFile(inProjectConfig)
		// Migrate PWA platform to custom+nodejs
		migratePWAToCustom(inProjectConfig, rawConf)
		// Re-read after potential migration
		rawConf = configs.ParseXmlFile(inProjectConfig)
		// For current dir config, get the merged project config for context
		migrateInProjectConfig(inProjectConfig, rawConf, rawConf)
	}
}

func migratePWAToCustom(configPath string, projectConf map[string]string) bool {
	platform := projectConf["platform"]
	if platform != "pwa" {
		return false
	}

	config := new(configs.ConfigLines)
	config.EnvFile = configPath
	config.ActiveScope = "default"
	if scope, ok := projectConf["activeScope"]; ok {
		config.ActiveScope = scope
	}

	config.Set("platform", "custom")
	config.Set("language", "nodejs")
	config.Set("nodejs/enabled", "true")
	config.Save()
	return true
}

func migrateProjectConfig(configPath string, projectConf map[string]string) {
	config := new(configs.ConfigLines)
	config.EnvFile = configPath
	config.ActiveScope = "default"
	if scope, ok := projectConf["activeScope"]; ok {
		config.ActiveScope = scope
	}

	changed := false

	// 1. Rename php/timezone -> timezone
	if tz, ok := projectConf["php/timezone"]; ok && tz != "" {
		if _, hasNew := projectConf["timezone"]; !hasNew {
			config.Set("timezone", tz)
			changed = true
		}
	}

	// 2. Add php/enabled for PHP-based projects
	platform := projectConf["platform"]
	language := projectConf["language"]
	if language == "" {
		language = "php"
	}

	switch platform {
	case "magento2", "shopware", "prestashop", "shopify":
		config.Set("php/enabled", "true")
		changed = true
	case "custom":
		if language == "php" {
			config.Set("php/enabled", "true")
			changed = true
		}
	}

	// 3. For custom PHP projects, rename nodejs/enabled -> php/nodejs/enabled
	if platform == "custom" && language == "php" {
		if nodeEnabled, ok := projectConf["nodejs/enabled"]; ok {
			if _, hasNew := projectConf["php/nodejs/enabled"]; !hasNew {
				config.Set("php/nodejs/enabled", nodeEnabled)
				changed = true
			}
		}
	}

	if changed {
		config.Save()
	}
}

func migrateInProjectConfig(configPath string, inProjectConf, projectConf map[string]string) {
	config := new(configs.ConfigLines)
	config.EnvFile = configPath
	config.ActiveScope = "default"
	if scope, ok := inProjectConf["activeScope"]; ok {
		config.ActiveScope = scope
	}

	changed := false

	// 1. Rename php/timezone -> timezone
	if tz, ok := inProjectConf["php/timezone"]; ok && tz != "" {
		if _, hasNew := inProjectConf["timezone"]; !hasNew {
			config.Set("timezone", tz)
			changed = true
		}
	}

	// 2. Add php/enabled for PHP-based projects
	platform := inProjectConf["platform"]
	if platform == "" {
		platform = projectConf["platform"]
	}
	language := inProjectConf["language"]
	if language == "" {
		language = projectConf["language"]
	}
	if language == "" {
		language = "php"
	}

	switch platform {
	case "magento2", "shopware", "prestashop", "shopify":
		config.Set("php/enabled", "true")
		changed = true
	case "custom":
		if language == "php" {
			config.Set("php/enabled", "true")
			changed = true
		}
	}

	// 3. For custom PHP projects, rename nodejs/enabled -> php/nodejs/enabled
	if platform == "custom" && language == "php" {
		if nodeEnabled, ok := inProjectConf["nodejs/enabled"]; ok {
			if _, hasNew := inProjectConf["php/nodejs/enabled"]; !hasNew {
				config.Set("php/nodejs/enabled", nodeEnabled)
				changed = true
			}
		}
	}

	if changed {
		config.Save()
	}
}
