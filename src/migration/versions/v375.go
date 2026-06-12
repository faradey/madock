package versions

import (
	"os"

	"github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/paths"
)

// V375 pins the historical DB credentials for projects created before the
// default DB user/password/database moved from "magento" to "db". New projects
// use the "db" defaults; existing projects keep their data and container volumes
// by writing their real creds into config.xml.
//
// Strategy: if a project config lacks an explicit value for db/user,
// db/password, or db/database, fill it in. For Magento 2 projects the real
// values are read from app/etc/env.php (the source of truth for what the running
// site actually uses); otherwise the legacy implicit default "magento" is used.
// Projects that already set those keys are left untouched.
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
		creds := legacyDbCreds(projectConf)
		backfillLegacyDbCreds(configPath, projectConf, creds)

		projectPath := projectConf["path"]
		if projectPath != "" {
			inProjectConfig := projectPath + "/.madock/config.xml"
			if paths.IsFileExist(inProjectConfig) {
				inProjectConf := configs.ParseXmlFile(inProjectConfig)
				inProjectConf = getConfigByScopeV366(inProjectConf, projectConf)
				backfillLegacyDbCreds(inProjectConfig, inProjectConf, creds)
			}
		}
	}
}

// legacyDbCreds resolves the db/user, db/password, db/database values to pin for
// a project. For Magento 2 it prefers the real credentials from env.php; it falls
// back to the historical implicit default "magento".
func legacyDbCreds(projectConf map[string]string) map[string]string {
	creds := map[string]string{"db/user": "magento", "db/password": "magento", "db/database": "magento"}
	if projectConf["platform"] == "magento2" {
		if projectPath := projectConf["path"]; projectPath != "" {
			if user, password, dbname, ok := ReadMagentoEnvDbCreds(projectPath + "/app/etc/env.php"); ok {
				creds["db/user"] = user
				creds["db/password"] = password
				creds["db/database"] = dbname
			}
		}
	}
	return creds
}

func backfillLegacyDbCreds(configPath string, projectConf, creds map[string]string) {
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
			config.Set(key, creds[key])
		}
	}
	config.Save()
}
