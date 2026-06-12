package configs

import (
	"bytes"
	_ "embed"
	"encoding/xml"
	"github.com/faradey/madock/v3/src/helper/logger"
	"github.com/faradey/madock/v3/src/helper/paths"
	"github.com/go-xmlfmt/xmlfmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

//go:embed config_defaults.xml
var defaultConfigXML []byte

var generalConfig map[string]string
var projectConfig map[string]string
var nameOfProject string

// ProjectNameResolver allows enterprise to customize how project names are derived.
// For example, to include git branch name for per-branch environments.
type ProjectNameResolver func() string

var projectNameResolver ProjectNameResolver

// SetProjectNameResolver sets a custom resolver for project name derivation.
func SetProjectNameResolver(r ProjectNameResolver) {
	projectNameResolver = r
}

// GetDefaultConfigXML returns the raw embedded config_defaults.xml bytes.
func GetDefaultConfigXML() []byte {
	return defaultConfigXML
}

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
	origGeneralConfig := make(map[string]string)

	// Always start with embedded defaults
	if len(defaultConfigXML) > 0 {
		origGeneralConfig = ParseXmlBytes(defaultConfigXML)
		origGeneralConfig = getConfigByScope(origGeneralConfig, "default")
	}

	// Overlay filesystem config.xml — file values win, embedded fills gaps
	configPath := paths.GetExecDirPath() + "/config.xml"
	if _, err := os.Stat(configPath); !os.IsNotExist(err) && err == nil {
		fileConfig := ParseXmlFile(configPath)
		fileConfig = getConfigByScope(fileConfig, "default")
		GeneralConfigMapping(origGeneralConfig, fileConfig)
		origGeneralConfig = fileConfig
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
	return GetProjectConfig(GetProjectName())
}

func SetCurrentProjectConfig(conf map[string]string) {
	projectConfig = conf
}

func GetProjectConfig(projectName string) map[string]string {
	if projectName == GetProjectName() {
		if len(projectConfig) == 0 {
			config := GetProjectConfigOnly(projectName)
			ConfigMapping(GetGeneralConfig(), config)
			projectConfig = config
		}
		return projectConfig
	} else {
		config := GetProjectConfigOnly(projectName)
		ConfigMapping(GetGeneralConfig(), config)
		return config
	}
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
	projectPath := ""
	if val, ok := activeConfig["path"]; ok {
		projectPath = val
	} else if projectName == GetProjectName() {
		// Safe only for the current project: CWD is its source directory.
		projectPath = paths.GetRunDirPath()
		activeConfig["path"] = projectPath
	} else {
		// For another project CWD is meaningless. Falling back to it would read
		// the current project's .madock/config.xml as if it belonged to
		// projectName (and persist a wrong `path` into its runtime config),
		// corrupting cross-project regenerators such as the shared proxy.conf
		// builder. Skip the release-side defaults; the runtime config alone
		// drives. GetProjectConfigInProject("") returns an empty map, so an
		// empty projectPath is safe here.
		warnMissingProjectPath(projectName)
	}
	defaultConfig := GetProjectConfigInProject(projectPath)
	activeProjectConfig := make(map[string]string)
	ConfigMapping(defaultConfig, activeProjectConfig)
	ConfigMapping(activeConfig, activeProjectConfig)
	return activeProjectConfig
}

// warnedMissingPath dedupes the missing-`path` warning so a single rebuild that
// reads a foreign project's config many times logs it only once per project.
var warnedMissingPath sync.Map

func warnMissingProjectPath(projectName string) {
	if _, loaded := warnedMissingPath.LoadOrStore(projectName, true); loaded {
		return
	}
	logger.Println("warning: project \"" + projectName + "\" has no 'path' key in runtime config; its release .madock/config.xml will not be merged (cross-project read)")
}

func GetCurrentProjectConfigPath(projectName string) string {
	if projectName == "" {
		projectName = GetProjectName()
	}
	return paths.MakeDirsByPath(paths.GetExecDirPath()+"/projects/"+projectName) + "/config.xml"
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
	if nameOfProject == "" && projectNameResolver != nil {
		nameOfProject = projectNameResolver()
	}
	if nameOfProject != "" {
		return nameOfProject
	}

	currentPath := canonicalProjectPath(paths.GetRunDirPath())
	suffix := ""
	for i := 2; i < 1000; i++ {
		nameOfProject = paths.GetRunDirName() + suffix
		configPath := paths.GetExecDirPath() + "/projects/" + nameOfProject + "/config.xml"
		if !paths.IsFileExist(configPath) {
			// No clash — this name is free for the current directory.
			break
		}
		projectConf := GetProjectConfigOnly(nameOfProject)
		stored, ok := projectConf["path"]
		if !ok {
			// Legacy project without `path` recorded — assume it owns this name.
			break
		}
		if canonicalProjectPath(stored) == currentPath {
			// Same project (just running it from a possibly different
			// path representation, e.g. /tmp vs /private/tmp on macOS).
			break
		}
		suffix = "-" + strconv.Itoa(i)
	}

	return nameOfProject
}

// canonicalProjectPath normalises a project path for comparison: trim
// whitespace and trailing slashes, then resolve symlinks when the path
// still exists on disk. On macOS `/tmp` is a symlink to `/private/tmp`,
// so two recordings of "the same" directory can differ textually; we
// want them to compare equal so GetProjectName doesn't auto-suffix a
// project that's actually the user's current one.
func canonicalProjectPath(p string) string {
	p = strings.TrimSpace(p)
	p = strings.TrimRight(p, "/")
	if resolved, err := filepath.EvalSymlinks(p); err == nil {
		return strings.TrimRight(resolved, "/")
	}
	return p
}

func IsProjectNameExists(name string) bool {
	currentPath := canonicalProjectPath(paths.GetRunDirPath())
	suffix := ""
	for i := 2; i < 1000; i++ {
		nameOfProject = paths.GetRunDirName() + suffix
		configPath := paths.GetExecDirPath() + "/projects/" + nameOfProject + "/config.xml"
		if !paths.IsFileExist(configPath) {
			break
		}
		projectConf := GetProjectConfigOnly(nameOfProject)
		stored, ok := projectConf["path"]
		if !ok {
			break
		}
		if canonicalProjectPath(stored) == currentPath {
			break
		}
		suffix = "-" + strconv.Itoa(i)
	}

	return false
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

func saveProjectConfig(configPath string, config map[string]string) bool {
	resultData := make(map[string]any)
	for key, value := range config {
		resultData[key] = value
	}
	resultMapData := SetXmlMap(resultData)
	w := &bytes.Buffer{}
	w.WriteString(xml.Header)
	encoder := xml.NewEncoder(w)
	defer func() { _ = encoder.Close() }()
	err := MarshalXML(resultMapData, encoder, "config")
	if err != nil {
		logger.Fatalln(err)
	}
	err = os.WriteFile(configPath, []byte(xmlfmt.FormatXML(w.String(), "", "    ", true)), ConfigFilePermissions)
	if err != nil {
		return false
	}
	return true
}

func SetScope(projectName, scope string) bool {
	configPath := GetCurrentProjectConfigPath(projectName)
	if _, err := os.Stat(configPath); os.IsNotExist(err) && err != nil {
		return false
	}

	config := ParseXmlFile(configPath)
	config["activeScope"] = scope
	return saveProjectConfig(configPath, config)
}

func AddScope(projectName, scope string) bool {
	configPath := GetCurrentProjectConfigPath(projectName)
	if _, err := os.Stat(configPath); os.IsNotExist(err) && err != nil {
		return false
	}

	config := ParseXmlFile(configPath)
	config["activeScope"] = scope
	config["scopes/"+scope] = ""
	return saveProjectConfig(configPath, config)
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
