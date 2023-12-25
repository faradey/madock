package configs

import (
	"log"
	"os"
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
	Save()
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

func (t *ConfigLines) Save() {
	err := os.WriteFile(t.EnvFile, []byte(strings.Join(t.Lines, "\n")), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
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
