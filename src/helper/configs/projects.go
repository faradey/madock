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

// defaultOverrides lets an edition disagree with a built-in default.
var defaultOverrides = map[string]string{}

// SetDefaultOverride changes what a setting defaults to, for editions whose
// answer differs from the community one.
//
// It applies after the embedded defaults and before the user's config.xml, which
// is the only layering that works: the edition gets to choose the default, and
// whoever edits the file still overrides it. Turning something back on must
// never require a different binary.
//
// The case it was written for is mailpit. It is a mail interceptor with no
// authentication, which is what a developer wants and the opposite of what a
// server wants — and madock-pro is the edition that runs on servers.
func SetDefaultOverride(key, value string) {
	defaultOverrides[key] = value
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

	// Then whatever this edition decided differs. Before the file below, so a
	// user who wants the community answer back can simply say so.
	for key, value := range defaultOverrides {
		origGeneralConfig[key] = value
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
			applyDerived(config)
			projectConfig = config
		}
		return projectConfig
	} else {
		config := GetProjectConfigOnly(projectName)
		ConfigMapping(GetGeneralConfig(), config)
		applyDerived(config)
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
// IsSamePath reports whether two recorded paths mean the same directory, with the
// same normalisation GetProjectName uses. Exported for callers that destroy things
// and must be sure the directory in front of them is the one they think it is.
func IsSamePath(a, b string) bool {
	return canonicalProjectPath(a) == canonicalProjectPath(b)
}

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

// Project states as ListProjectsIn reports them.
const (
	// ProjectOk — the directory the project was set up in is still there.
	ProjectOk = "ok"
	// ProjectMissingSource — the recorded path points at a directory that is
	// gone. The entry keeps being resolved: it holds its ports, and the shared
	// proxy keeps a server block routing its hosts at containers that cannot
	// exist. Two such entries were found on this machine, both still in the
	// generated proxy configuration.
	ProjectMissingSource = "missing-source"
	// ProjectNoPath — a legacy entry from before the path was recorded. The
	// self-heal in IsHasConfig backfills it, but only when madock runs from the
	// project's own directory, which is exactly what nobody does for a project
	// they have forgotten about.
	ProjectNoPath = "no-path"
)

// ProjectEntry is one registry entry and what is true about it.
type ProjectEntry struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	State string `json:"state"`
}

// ListProjectsIn reads the registry under an installation directory.
//
// A project is a directory under projects/ that has a config.xml — not merely a
// directory. That distinction matters: aruntime/projects/ also holds `composer`
// and `ssh`, which are shared support directories rather than projects, and
// anything that walks names instead of configurations picks them up.
//
// Takes the directory rather than asking paths for it, so a test can build a
// registry in a temporary one.
func ListProjectsIn(execDir string) []ProjectEntry {
	entries, err := os.ReadDir(filepath.Join(execDir, "projects"))
	if err != nil {
		return nil
	}

	var out []ProjectEntry
	for _, entry := range entries {
		// Stat rather than entry.IsDir(): os.ReadDir answers about the entry, and a
		// symlink to a directory says false. Registry entries are symlinks wherever
		// a project was set up from a temporary checkout, and on a cluster VM with
		// four such projects this command answered "No projects are registered" —
		// the one wrong answer that reads as good news. A broken symlink fails the
		// Stat and is skipped, which is right for an entry pointing at nothing.
		info, statErr := os.Stat(filepath.Join(execDir, "projects", entry.Name()))
		if statErr != nil || !info.IsDir() {
			continue
		}
		configPath := filepath.Join(execDir, "projects", entry.Name(), "config.xml")
		if !paths.IsFileExist(configPath) {
			continue
		}

		raw := ParseXmlFile(configPath)
		path := strings.TrimSpace(getConfigByScope(raw, "default")["path"])

		state := ProjectOk
		switch {
		case path == "":
			state = ProjectNoPath
		case !paths.IsFileExist(path):
			state = ProjectMissingSource
		}

		out = append(out, ProjectEntry{Name: entry.Name(), Path: path, State: state})
	}

	return out
}

// ListProjects reads the registry of the installation in use.
func ListProjects() []ProjectEntry {
	return ListProjectsIn(paths.GetExecDirPath())
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
