package versions

import (
	"os"

	"github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/paths"
)

// V384 repairs Magento 2 projects whose stored DB credentials drifted away from
// app/etc/env.php. An earlier run of V375 backfilled the implicit default
// "magento" for projects that actually run on different credentials (e.g. the
// "db" default with a generated password), which made db:export/db:import target
// a non-existent database. env.php is the source of truth for what the running
// site uses, so this migration realigns db/user, db/password and db/database to
// the env.php values. Projects whose config already matches env.php are left
// untouched; non-Magento projects and projects without a readable env.php are
// skipped.
func V384() {
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
		if projectConf["platform"] != "magento2" {
			continue
		}
		projectPath := projectConf["path"]
		if projectPath == "" {
			continue
		}

		user, password, dbname, ok := ReadMagentoEnvDbCreds(projectPath + "/app/etc/env.php")
		if !ok {
			continue
		}
		desired := map[string]string{"db/user": user, "db/password": password, "db/database": dbname}

		alignDbCredsToEnv(configPath, projectConf, desired)

		inProjectConfig := projectPath + "/.madock/config.xml"
		if paths.IsFileExist(inProjectConfig) {
			inProjectConf := configs.ParseXmlFile(inProjectConfig)
			inProjectConf = getConfigByScopeV366(inProjectConf, projectConf)
			alignDbCredsToEnv(inProjectConfig, inProjectConf, desired)
		}
	}
}

func alignDbCredsToEnv(configPath string, projectConf, desired map[string]string) {
	config := new(configs.ConfigLines)
	config.EnvFile = configPath
	config.ActiveScope = "default"
	if scope, ok := projectConf["activeScope"]; ok {
		config.ActiveScope = scope
	}

	changed := false
	for key, want := range desired {
		if projectConf[key] != want {
			config.Set(key, want)
			changed = true
		}
	}
	if changed {
		config.Save()
	}
}
