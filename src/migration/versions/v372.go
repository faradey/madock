package versions

import (
	"os"

	"github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/paths"
)

// V372 is a follow-up to V366 that catches projects upgraded from versions
// in the 3.6.7..3.7.1 range. The original V366 only ran when oldAppVersion
// was strictly less than 3.6.7, leaving projects that were created or
// upgraded inside that window without php/enabled, which breaks
// docker-compose because nginx and php_without_xdebug depend on the php
// service. V372 also adds woocommerce and shopify to the platform list.
func V372() {
	phpPlatforms := map[string]bool{
		"magento2":    true,
		"shopware":    true,
		"prestashop":  true,
		"woocommerce": true,
		"shopify":     true,
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
