package configs

import (
	"log"
	"os"
	"os/user"
	"regexp"
	"runtime"
	"strings"

	"github.com/faradey/madock/src/paths"
)

type ConfigLines struct {
	Lines   []string
	EnvFile string
	IsEnv   bool
}

type ConfigLinesInterface interface {
	AddLine(name, value string)
	AddOrSetLine(name, value string)
	AddEmptyLine()
	AddRawLine(value string)
	SaveLines()
}

func (t *ConfigLines) AddLine(name, value string) {
	t.Lines = append(t.Lines, name+"="+value)
}

func (t *ConfigLines) AddOrSetLine(name, value string) {
	if !t.IsEnv {
		t.Lines = append(t.Lines, name+"="+value)
	} else {
		SetParam(t.EnvFile, name, value)
	}
}

func (t *ConfigLines) AddEmptyLine() {
	t.Lines = append(t.Lines, "")
}

func (t *ConfigLines) AddRawLine(value string) {
	t.Lines = append(t.Lines, value)
}

func (t *ConfigLines) SaveLines() {
	err := os.WriteFile(t.EnvFile, []byte(strings.Join(t.Lines, "\n")), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
}

func IsHasConfig(projectName string) bool {
	PrepareDirsForProject(projectName)
	envFile := paths.GetExecDirPath() + "/projects/" + projectName + "/env.txt"
	if _, err := os.Stat(envFile); !os.IsNotExist(err) {
		return true
	}

	return false
}

func IsHasNotConfig() bool {
	envFile := paths.GetExecDirPath() + "/projects/" + GetProjectName() + "/env.txt"
	if _, err := os.Stat(envFile); os.IsNotExist(err) {
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
	if len(targetConf) > 0 && len(mainConf) > 0 {
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

	str = strings.Replace(str, "{{{OSARCH}}}", osArch, -1)

	usr, err := user.Current()
	if err == nil {
		str = strings.Replace(str, "{{{UID}}}", usr.Uid, -1)
		str = strings.Replace(str, "{{{UNAME}}}", usr.Username, -1)
		str = strings.Replace(str, "{{{GUID}}}", usr.Gid, -1)
		gr, _ := user.LookupGroupId(usr.Gid)
		str = strings.Replace(str, "{{{UGROUP}}}", gr.Name, -1)
	} else {
		log.Fatal(err)
	}

	r := regexp.MustCompile("(?ism)<<<iftrue>>>(.*?)<<<endif>>>")
	str = r.ReplaceAllString(str, "$1")
	r = regexp.MustCompile("(?ism)<<<iffalse>>>.*?<<<endif>>>")
	str = r.ReplaceAllString(str, "")

	var onlyHosts []string

	hosts := strings.Split(projectConf["HOSTS"], " ")
	if len(hosts) > 0 {
		for _, hostAndStore := range hosts {
			onlyHosts = append(onlyHosts, "- \""+strings.Split(hostAndStore, ":")[0]+":172.17.0.1\"")
		}

		if len(onlyHosts) > 0 {
			str = strings.Replace(str, "{{{HOST_GATEWAYS}}}", strings.Join(onlyHosts, "\n"), -1)
		}
	}
	return str
}

func IsOption(name string) bool {
	upperName := strings.ToUpper(name)
	projectConf := GetCurrentProjectConfig()

	for key := range projectConf {
		if key == upperName {
			return true
		}
	}

	log.Fatalln("The option \"" + name + "\" doesn't exist.")

	return false
}
