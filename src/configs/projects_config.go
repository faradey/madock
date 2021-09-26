package configs

import (
	"github.com/faradey/madock/src/paths"
	"log"
	"os"
)

func GetGeneralConfig() map[string]string {
	configPath := paths.GetExecDirPath() + "/projects/config"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err != nil {
			configPath = paths.GetExecDirPath() + "/projects/config.sample"
			if _, err = os.Stat(configPath); os.IsNotExist(err) {
				if err != nil {
					log.Fatal(err)
				}
			}
		}
	}

	return ParseFile(configPath)
}

func GetProjectConfig() map[string]string {
	configPath := paths.GetExecDirPath() + "/projects/" + paths.GetRunDirName() + "/env"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err != nil {
			log.Fatal(err)
		}
	}

	return ParseFile(configPath)
}
