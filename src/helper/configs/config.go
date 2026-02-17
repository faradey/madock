package configs

import (
	"bytes"
	"encoding/xml"
	"log"
	"net"
	"os"
	"os/user"
	"runtime"
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

	for key, value := range data {
		resultData["scopes/"+activeScope+"/"+key] = value
	}

	// Encrypt secret values before writing to XML
	for key, value := range resultData {
		if strVal, ok := value.(string); ok {
			resultData[key] = encryptIfSecret(key, strVal)
		}
	}

	resultMapData := SetXmlMap(resultData)
	w := &bytes.Buffer{}
	w.WriteString(xml.Header)
	err := MarshalXML(resultMapData, xml.NewEncoder(w), "config")
	if err != nil {
		logger.Fatalln(err)
	}

	err = os.WriteFile(file, []byte(xmlfmt.FormatXML(w.String(), "", "    ", true)), ConfigFilePermissions)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
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
	if !paths.IsFileExist(paths.GetExecDirPath()+"/projects/"+projectName+"/config.xml") && paths.IsFileExist(paths.GetRunDirPath()+"/.madock/config.xml") {
		err := paths.Copy(paths.GetRunDirPath()+"/.madock/config.xml", paths.GetExecDirPath()+"/projects/"+projectName+"/config.xml")
		if err != nil {
			logger.Println(err)
			return false
		}
		SetParam(projectName, "path", paths.GetRunDirPath(), "default", MadockLevelConfigCode)
	}
	if paths.IsFileExist(paths.GetExecDirPath()+"/projects/"+projectName+"/config.xml") || paths.IsFileExist(paths.GetRunDirPath()+"/.madock/config.xml") {
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

func ReplaceConfigValue(projectName, str string) string {
	projectConf := GetProjectConfig(projectName)
	osArch := runtime.GOARCH
	arches := map[string]string{"arm64": "aarch64"}

	if arch, ok := arches[osArch]; ok {
		osArch = arch
	} else {
		osArch = "x86-64"
	}

	// Compute db/use_default_auth_plugin based on MySQL/MariaDB version
	// MySQL 8.4+ removed --default-authentication-plugin option
	dbRepo := strings.ToLower(projectConf["db/repository"])
	dbVersion := projectConf["db/version"]
	useDefaultAuthPlugin := "true"
	if dbRepo == "mysql" && CompareVersions(dbVersion, "8.4") >= 0 {
		useDefaultAuthPlugin = "false"
	}
	projectConf["db/use_default_auth_plugin"] = useDefaultAuthPlugin

	for key, val := range projectConf {
		str = strings.Replace(str, "{{{"+key+"}}}", val, -1)
	}

	str = strings.Replace(str, "{{{os/arch}}}", osArch, -1)

	usr, err := user.Current()
	if err == nil {
		str = strings.Replace(str, "{{{os/user/uid}}}", usr.Uid, -1)
		str = strings.Replace(str, "{{{os/user/name}}}", usr.Username, -1)
		str = strings.Replace(str, "{{{os/user/guid}}}", usr.Gid, -1)
		gr, err := user.LookupGroupId(usr.Gid)
		if err == nil && gr != nil {
			str = strings.Replace(str, "{{{os/user/ugroup}}}", gr.Name, -1)
		} else {
			str = strings.Replace(str, "{{{os/user/ugroup}}}", usr.Username, -1)
		}
	} else {
		logger.Fatal(err)
	}

	// Process conditionals with proper nesting support
	str = processConditionals(str)

	var onlyHosts []string

	hosts := GetHosts(projectConf)
	if len(hosts) > 0 {
		for _, host := range hosts {
			onlyHosts = append(onlyHosts, "- \""+host["name"]+":"+GetOutboundIP()+"\"")
		}
	}

	str = strings.Replace(str, "{{{nginx/host_gateways}}}", strings.Join(onlyHosts, "\n      "), -1)
	return str
}

// processConditionals handles nested <<<if...>>>...<<<endif>>> blocks
func processConditionals(str string) string {
	for {
		// Find the first <<<if...>>> tag
		ifStart := strings.Index(str, "<<<if")
		if ifStart == -1 {
			break // No more conditionals
		}

		// Find the end of the opening tag (>>>)
		tagEnd := strings.Index(str[ifStart:], ">>>")
		if tagEnd == -1 {
			break // Malformed tag
		}
		tagEnd += ifStart + 3 // Position after >>>

		// Safety check
		if tagEnd > len(str) {
			break
		}

		// Extract the condition (between <<<if and >>>)
		condition := str[ifStart+5 : tagEnd-3]

		// Find the matching <<<endif>>> (accounting for nesting)
		endifPos := findMatchingEndif(str, tagEnd)
		if endifPos == -1 {
			break // No matching endif
		}

		// Extract content between opening and closing tags
		content := str[tagEnd:endifPos]

		// Evaluate condition
		shouldKeep := evaluateCondition(condition)

		// Calculate end position after <<<endif>>>
		endPos := endifPos + 11 // 11 = len("<<<endif>>>")
		if endPos > len(str) {
			endPos = len(str)
		}

		// Build the replacement
		var replacement string
		if shouldKeep {
			// Recursively process nested conditionals in the content
			replacement = processConditionals(content)
			str = str[:ifStart] + replacement + str[endPos:]
		} else {
			// When removing, also remove leading whitespace on the same line
			lineStart := ifStart
			for lineStart > 0 && str[lineStart-1] != '\n' {
				lineStart--
			}
			// Check if line only has whitespace before <<<if
			prefix := str[lineStart:ifStart]
			if strings.TrimSpace(prefix) == "" {
				// Remove from line start (including leading whitespace)
				// Also remove trailing newline if the line becomes empty
				afterEnd := str[endPos:]
				if len(afterEnd) > 0 && afterEnd[0] == '\n' {
					endPos++ // Skip the newline after <<<endif>>>
				}
				str = str[:lineStart] + str[endPos:]
			} else {
				// There's content before <<<if on this line, just remove the block
				str = str[:ifStart] + str[endPos:]
			}
		}
	}

	return str
}

// findMatchingEndif finds the position of <<<endif>>> that matches the opening tag
// accounting for nested conditionals
func findMatchingEndif(str string, startPos int) int {
	depth := 1
	pos := startPos

	for depth > 0 && pos < len(str) {
		nextIf := strings.Index(str[pos:], "<<<if")
		nextEndif := strings.Index(str[pos:], "<<<endif>>>")

		if nextEndif == -1 {
			return -1 // No matching endif found
		}

		// Check which comes first
		if nextIf != -1 && nextIf < nextEndif {
			// Found another <<<if before <<<endif>>>
			depth++
			pos += nextIf + 5 // Move past <<<if
		} else {
			// Found <<<endif>>>
			depth--
			if depth == 0 {
				return pos + nextEndif
			}
			pos += nextEndif + 11 // Move past <<<endif>>>
		}
	}

	return -1
}

// evaluateCondition checks if the condition should evaluate to true
func evaluateCondition(condition string) bool {
	// Trim whitespace
	condition = strings.TrimSpace(condition)

	// Empty condition = false
	if condition == "" {
		return false
	}

	// Contains unprocessed placeholder = false
	if strings.Contains(condition, "{{{") {
		return false
	}

	// Contains "false" anywhere = false
	if strings.Contains(strings.ToLower(condition), "false") {
		return false
	}

	// Contains only "true" (possibly repeated) = true
	// Remove all "true" and whitespace, if nothing left = true
	cleaned := strings.ReplaceAll(strings.ToLower(condition), "true", "")
	cleaned = strings.TrimSpace(cleaned)
	if cleaned == "" {
		return true
	}

	// Any other non-empty value without false = true
	return true
}

func GetOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "172.17.0.1"
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP.String()
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
