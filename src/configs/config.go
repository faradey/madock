package configs

import (
	"bufio"
	"fmt"
	"github.com/faradey/madock/src/cli/fmtc"
	"github.com/faradey/madock/src/paths"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"runtime"
	"strings"
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
	err := ioutil.WriteFile(t.EnvFile, []byte(strings.Join(t.Lines, "\n")), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
}

func IsHasConfig() bool {
	paths.PrepareDirsForProject()
	projectName := paths.GetRunDirName()
	envFile := paths.GetExecDirPath() + "/projects/" + projectName + "/env.txt"
	if _, err := os.Stat(envFile); !os.IsNotExist(err) {
		fmtc.WarningLn("File env is already exist in project " + projectName)
		fmt.Println("Do you want to continue? (y/N)")
		fmt.Print("> ")

		buf := bufio.NewReader(os.Stdin)
		sentence, err := buf.ReadBytes('\n')
		selected := strings.TrimSpace(string(sentence))
		if err != nil {
			log.Fatal(err)
		} else {
			if selected != "y" {
				log.Fatal("Exit")
			}
		}

		return true
	}

	return false
}

func IsHasNotConfig() bool {
	envFile := paths.GetExecDirPath() + "/projects/" + paths.GetRunDirName() + "/env.txt"
	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		return true
	}
	return false
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

	return str
}
