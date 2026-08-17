package configs

import (
	"bytes"
	"encoding/xml"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/faradey/madock/v3/src/helper/logger"
	"github.com/faradey/madock/v3/src/helper/paths"
	"github.com/go-xmlfmt/xmlfmt"
)

// ConfigFilePermissions controls the file mode used when writing config XML files.
// Default is 0644 (owner read/write, others read). Enterprise can override via SetConfigFilePermissions.
var ConfigFilePermissions os.FileMode = 0644

// SetConfigFilePermissions overrides the file mode for config XML files.
func SetConfigFilePermissions(perm os.FileMode) {
	ConfigFilePermissions = perm
}

type ConfigLines struct {
	Lines       map[string]string
	EnvFile     string
	ActiveScope string
}

type ConfigLinesInterface interface {
	Set(name, value string)
	Save()
}

func (t *ConfigLines) Save() {
	SaveInFile(t.EnvFile, t.Lines, t.ActiveScope)
}

func SaveInFile(file string, data map[string]string, activeScope string) {
	resultData := make(map[string]interface{})
	if paths.IsFileExist(file) {
		config := ParseXmlFile(file)
		for key, value := range config {
			resultData[key] = value
		}
	}

	// If the incoming data redefines hosts, treat nginx/hosts/* as a replace-group:
	// drop existing host entries for this scope first, so a removed/recoded domain
	// doesn't linger (additive merge would keep a stale code like base2 next to the
	// new set -> "Duplicate domains").
	hasHosts := false
	for key := range data {
		if strings.HasPrefix(key, "nginx/hosts/") {
			hasHosts = true
			break
		}
	}
	if hasHosts {
		prefix := "scopes/" + activeScope + "/nginx/hosts/"
		for key := range resultData {
			if strings.HasPrefix(key, prefix) {
				delete(resultData, key)
			}
		}
	}

	for key, value := range data {
		resultData["scopes/"+activeScope+"/"+key] = value
	}

	// Encrypt secret values before writing to XML
	for key, value := range resultData {
		if strVal, ok := value.(string); ok {
			resultData[key] = encryptIfSecret(key, strVal)
		}
	}

	err := os.WriteFile(file, []byte(RenderXml(resultData)), ConfigFilePermissions)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
}

// RenderXml turns a flat key map back into the config file format.
//
// Shared with the migrations, which need to write a config file without going
// through SaveInFile's scope merging: a rename moves a key rather than setting
// one, and merging would leave both spellings in the file.
func RenderXml(data map[string]interface{}) string {
	w := &bytes.Buffer{}
	w.WriteString(xml.Header)
	if err := MarshalXML(SetXmlMap(data), xml.NewEncoder(w), "config"); err != nil {
		logger.Fatalln(err)
	}
	// Trimmed at the front and terminated at the end.
	//
	// The formatter emits a leading blank line and no trailing newline, and
	// these files live in somebody's repository: every write then shows up as
	// two spurious line changes on top of the real one. Cheap to fix, and it is
	// the difference between a diff that can be read and a diff that gets
	// waved through.
	return strings.TrimLeft(xmlfmt.FormatXML(w.String(), "", "    ", true), "\r\n") + "\n"
}

func (t *ConfigLines) Set(name, value string) {
	if t.Lines == nil {
		t.Lines = make(map[string]string)
	}
	if name == "hosts" {
		hosts := strings.Split(value, " ")
		for key, host := range hosts {
			splitHost := strings.Split(host, ":")
			runCode := "base"
			if key > 0 {
				runCode += strconv.Itoa(key + 1)
			}
			if len(splitHost) > 1 {
				runCode = splitHost[1]
			}
			t.Lines["nginx/hosts/"+runCode+"/name"] = splitHost[0]
		}
	} else {
		t.Lines[name] = value
	}
}

func IsHasConfig(projectName string) bool {
	if projectName == "" {
		projectName = GetProjectName()
	}
	PrepareDirsForProject(projectName)
	runtimeConfigPath := paths.GetExecDirPath() + "/projects/" + projectName + "/config.xml"
	inProjectConfigExists := paths.IsFileExist(paths.GetRunDirPath() + "/.madock/config.xml")
	if !paths.IsFileExist(runtimeConfigPath) && inProjectConfigExists {
		err := paths.Copy(paths.GetRunDirPath()+"/.madock/config.xml", runtimeConfigPath)
		if err != nil {
			logger.Println(err)
			return false
		}
		SetParam(projectName, "path", paths.GetRunDirPath(), "default", MadockLevelConfigCode)
	} else if inProjectConfigExists && paths.IsFileExist(runtimeConfigPath) {
		// Self-heal: legacy projects (bootstrapped before `path` was recorded) and
		// runtime configs placed by a deploy keep a runtime config without a
		// `path` key, which breaks cross-project regenerators (the shared
		// proxy.conf builder reads a foreign project and can't find its source).
		// We are in the project's own source dir here (CWD has its .madock), so
		// GetRunDirPath() is authoritative — backfill the missing key once.
		raw := ParseXmlFile(runtimeConfigPath)
		if getConfigByScope(raw, "default")["path"] == "" {
			SetParam(projectName, "path", paths.GetRunDirPath(), "default", MadockLevelConfigCode)
		}
	}
	if paths.IsFileExist(runtimeConfigPath) || inProjectConfigExists {
		return true
	}

	return false
}

func GeneralConfigMapping(mainConf map[string]string, targetConf map[string]string) {
	if len(mainConf) > 0 {
		for index, val := range mainConf {
			if v, ok := targetConf[index]; !ok || v == "" {
				targetConf[index] = val
			}
		}
	}
}

func ConfigMapping(mainConf map[string]string, targetConf map[string]string) {
	if len(mainConf) > 0 {
		for index, val := range mainConf {
			if _, ok := targetConf[index]; !ok {
				targetConf[index] = val
			}
		}
	}
}

// CompareVersions compares two version strings (e.g., "8.4" vs "8.3.1")
// Returns: 1 if v1 > v2, -1 if v1 < v2, 0 if equal
func CompareVersions(v1, v2 string) int {
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	maxLength := len(parts1)
	if len(parts2) > maxLength {
		maxLength = len(parts2)
	}

	for i := 0; i < maxLength; i++ {
		var p1, p2 int

		if i < len(parts1) {
			p1, _ = strconv.Atoi(parts1[i])
		}
		if i < len(parts2) {
			p2, _ = strconv.Atoi(parts2[i])
		}

		if p1 > p2 {
			return 1
		} else if p1 < p2 {
			return -1
		}
	}

	return 0
}

func IsOption(name string) bool {
	if strings.Contains(name, "/hosts/") {
		return true
	}

	// A derived key is recomputed on every read, so accepting a value for it
	// would store something nothing ever looks at.
	if source, derived := IsDerived(name); derived {
		logger.Fatalln("The option \"" + name + "\" is derived from \"" + source + "\" and cannot be set on its own. Set \"" + source + "\" instead.")
	}
	for key := range GetCurrentProjectConfig() {
		if key == name {
			return true
		}
	}

	logger.Fatalln("The option \"" + name + "\" doesn't exist.")

	return false
}

func GetHosts(data map[string]string) []map[string]string {
	var hosts []map[string]string
	sortedKeys := SortMap(data)
	for _, key := range sortedKeys {
		if strings.Contains(key, "/hosts/") && data[key] != "" {
			items := strings.Split(key, "/")
			hosts = append(hosts, map[string]string{"name": data[key], "code": items[len(items)-2]})
		}
	}

	return hosts
}

func GetCommands(data map[string]string) map[string]map[string]string {
	var commands map[string]map[string]string
	commands = make(map[string]map[string]string)
	sortedKeys := SortMap(data)
	for _, key := range sortedKeys {
		if strings.Contains(key, "custom_commands/") && data[key] != "" {
			items := strings.Split(key, "/")
			commandName := items[1]
			code := ""
			if strings.Contains(key, "/alias") {
				code = "alias"
			} else if strings.Contains(key, "/origin") {
				code = "origin"
			}

			if _, ok := commands[commandName]; !ok {
				commands[commandName] = make(map[string]string)
			}
			commands[commandName][code] = data[key]
		}
	}

	return commands
}

func SortMap(data map[string]string) []string {
	keys := make([]string, 0, len(data))

	for k := range data {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	return keys
}
