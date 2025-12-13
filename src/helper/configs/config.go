package configs

import (
	"bytes"
	"encoding/xml"
	"github.com/faradey/madock/src/helper/logger"
	"github.com/faradey/madock/src/helper/paths"
	"github.com/go-xmlfmt/xmlfmt"
	"log"
	"net"
	"os"
	"os/user"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

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
	resultMapData := SetXmlMap(resultData)
	w := &bytes.Buffer{}
	w.WriteString(xml.Header)
	err := MarshalXML(resultMapData, xml.NewEncoder(w), "config")
	if err != nil {
		logger.Fatalln(err)
	}

	err = os.WriteFile(file, []byte(xmlfmt.FormatXML(w.String(), "", "    ", true)), 0755)
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

	// Process conditionals:
	// 1. Keep content if condition contains only "true"
	r := regexp.MustCompile("(?ism)<<<if(true\\s*)+>>>(.*?)<<<endif>>>")
	str = r.ReplaceAllString(str, "$2")
	// 2. Remove content if condition contains "false"
	r = regexp.MustCompile("(?ism)<<<if.*?(false\\s*)+.*?>>>.*?<<<endif>>>")
	str = r.ReplaceAllString(str, "")
	// 3. Remove unprocessed conditionals (placeholders not replaced = treat as false)
	r = regexp.MustCompile("(?ism)<<<if\\{\\{\\{[^>]+>>>.*?<<<endif>>>")
	str = r.ReplaceAllString(str, "")

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

func GetOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "172.17.0.1"
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP.String()
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
