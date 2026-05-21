package versions

import (
	"os"

	"github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/paths"
)

// V375 preserves legacy "magento" DB credentials for projects created
// before the default DB user/password/database moved from "magento" to "db".
// New projects use the "db" defaults; existing projects keep their data and
// container volumes by pinning the historical creds in their config.xml.
//
// Strategy: if a project config lacks an explicit value for db/user,
// db/password, or db/database, write "magento" — that was the implicit value
// inherited from the embedded defaults before this release. Projects that
// already set those keys are left untouched.
func V375() {
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
		backfillLegacyDbCreds(configPath, projectConf)

		projectPath := projectConf["path"]
		if projectPath != "" {
			inProjectConfig := projectPath + "/.madock/config.xml"
			if paths.IsFileExist(inProjectConfig) {
				inProjectConf := configs.ParseXmlFile(inProjectConfig)
				inProjectConf = getConfigByScopeV366(inProjectConf, projectConf)
				backfillLegacyDbCreds(inProjectConfig, inProjectConf)
			}
		}
	}
}

func backfillLegacyDbCreds(configPath string, projectConf map[string]string) {
	keys := []string{"db/user", "db/password", "db/database"}
	missing := false
	for _, key := range keys {
		if _, has := projectConf[key]; !has {
			missing = true
			break
		}
	}
	if !missing {
		return
	}

	config := new(configs.ConfigLines)
	config.EnvFile = configPath
	config.ActiveScope = "default"
	if scope, ok := projectConf["activeScope"]; ok {
		config.ActiveScope = scope
	}

	for _, key := range keys {
		if _, has := projectConf[key]; !has {
			config.Set(key, "magento")
		}
	}
	config.Save()
}
