package configs

import (
	"bytes"
	"encoding/xml"
	"github.com/faradey/madock/src/helper/paths"
	"github.com/go-xmlfmt/xmlfmt"
	"log"
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

	if _, err := os.Stat(file); os.IsNotExist(err) && err != nil {
		log.Fatalln(err)
	}
	config := ParseXmlFile(file)
	resultData := make(map[string]interface{})
	for key, value := range config {
		resultData[key] = value
	}
	for key, value := range data {
		resultData["scopes/"+activeScope+"/"+key] = value
	}
	resultMapData := SetXmlMap(resultData)
	w := &bytes.Buffer{}
	w.WriteString(xml.Header)
	err := MarshalXML(resultMapData, xml.NewEncoder(w), "config")
	if err != nil {
		log.Fatalln(err)
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
	PrepareDirsForProject(projectName)
	if paths.IsFileExist(paths.GetExecDirPath() + "/projects/" + projectName + "/config.xml") {
		return true
	}

	return false
}

func IsHasNotConfig() bool {
	if !paths.IsFileExist(paths.GetExecDirPath() + "/projects/" + GetProjectName() + "/config.xml") {
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

func ReplaceConfigValue(str string) string {
	projectConf := GetCurrentProjectConfig()
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
		gr, _ := user.LookupGroupId(usr.Gid)
		str = strings.Replace(str, "{{{os/user/ugroup}}}", gr.Name, -1)
	} else {
		log.Fatal(err)
	}

	r := regexp.MustCompile("(?ism)<<<iftrue>>>(.*?)<<<endif>>>")
	str = r.ReplaceAllString(str, "$1")
	r = regexp.MustCompile("(?ism)<<<iffalse>>>.*?<<<endif>>>")
	str = r.ReplaceAllString(str, "")

	var onlyHosts []string

	hosts := GetHosts(projectConf)
	if len(hosts) > 0 {
		for _, host := range hosts {
			onlyHosts = append(onlyHosts, "- \""+host["name"]+":172.17.0.1\"")
		}
	}

	str = strings.Replace(str, "{{{nginx/host_gateways}}}", strings.Join(onlyHosts, "\n      "), -1)
	return str
}

func IsOption(name string) bool {
	for key := range GetCurrentProjectConfig() {
		if key == name {
			return true
		}
	}

	log.Fatalln("The option \"" + name + "\" doesn't exist.")

	return false
}

func GetHosts(data map[string]string) []map[string]string {
	var hosts []map[string]string
	sortedKeys := SortMap(data)
	for _, key := range sortedKeys {
		if strings.Contains(key, "/hosts/") && data[key] != "" {
			items := strings.Split(key, "/")
			hosts = append(hosts, map[string]string{"name": data[key], "code": items[len(items)-1]})
		}
	}

	return hosts
}

func SortMap(data map[string]string) []string {
	keys := make([]string, 0, len(data))

	for k := range data {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	return keys
}
