package versions

import (
	"os"
	"strings"

	"github.com/faradey/madock/v4/src/helper/configs"
	"github.com/faradey/madock/v4/src/helper/paths"
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
		// Scoped before the lookup, and the omission was expensive.
		//
		// ParseXmlFile returns keys as they sit in the file, so a setting in
		// the default scope arrives as "scopes/default/language". Asking for
		// "language" therefore never found one, the guard always passed, and
		// this wrote php over whatever the project actually was — measured on
		// a nodejs project, where it sent `madock cli` into a php container
		// that does not exist and cost half an hour looking for a defect in
		// the service resolver.
		//
		// It only fires when the migrations run at all, which is why it hid
		// for so long: an installation with a current recorded version never
		// gets here, and a fresh one — the only safe way to try a new build —
		// gets here every time.
		rawConf := scopedV320(configs.ParseXmlFile(inProjectConfig))
		if language, ok := rawConf["language"]; !ok || language == "" {
			config := new(configs.ConfigLines)
			config.EnvFile = inProjectConfig
			config.ActiveScope = "default"
			if scope, ok := rawConf["activeScope"]; ok {
				config.ActiveScope = scope
			}
			config.Set("language", "php")
			config.Save()
		}
	}
}

// scopedV320 strips the scope prefix so a setting can be looked up by its own
// name. Frozen here rather than shared: a migration has to keep meaning what it
// meant when it was written, and the general helper is free to change.
func scopedV320(rawConf map[string]string) map[string]string {
	activeScope := "default"
	if scope, ok := rawConf["activeScope"]; ok && scope != "" {
		activeScope = scope
	}

	prefix := "scopes/" + activeScope + "/"
	result := make(map[string]string, len(rawConf))
	for key, value := range rawConf {
		if strings.HasPrefix(key, prefix) {
			result[strings.TrimPrefix(key, prefix)] = value
		}
	}
	if scope, ok := rawConf["activeScope"]; ok {
		result["activeScope"] = scope
	}

	return result
}
