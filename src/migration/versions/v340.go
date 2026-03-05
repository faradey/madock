package versions

import (
	"os"

	"github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/paths"
)

// V340 migrates config for PostgreSQL and MongoDB support:
// 1. Adds db/type based on db/repository for existing projects
func V340() {
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
		migrateDbType(configPath, projectConf)

		// Also check in-project config
		projectPath := projectConf["path"]
		if projectPath != "" {
			inProjectConfig := projectPath + "/.madock/config.xml"
			if paths.IsFileExist(inProjectConfig) {
				inProjectConf := configs.ParseXmlFile(inProjectConfig)
				inProjectConf = getConfigByScopeV340(inProjectConf, projectConf)
				migrateDbType(inProjectConfig, inProjectConf)
			}
		}
	}

	// Also check current directory for .madock/config.xml
	currentPath := paths.GetRunDirPath()
	inProjectConfig := currentPath + "/.madock/config.xml"
	if paths.IsFileExist(inProjectConfig) {
		rawConf := configs.ParseXmlFile(inProjectConfig)
		rawConf = getConfigByScopeV340(rawConf, rawConf)
		migrateDbType(inProjectConfig, rawConf)
	}
}

func migrateDbType(configPath string, projectConf map[string]string) {
	if _, hasType := projectConf["db/type"]; hasType {
		return
	}

	dbType := configs.GetDbType(projectConf)

	config := new(configs.ConfigLines)
	config.EnvFile = configPath
	config.ActiveScope = "default"
	if scope, ok := projectConf["activeScope"]; ok {
		config.ActiveScope = scope
	}

	config.Set("db/type", dbType)
	config.Save()
}

func getConfigByScopeV340(rawConf, fallbackConf map[string]string) map[string]string {
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
	// Inherit activeScope for downstream use
	if scope, ok := fallbackConf["activeScope"]; ok {
		result["activeScope"] = scope
	}
	return result
}
