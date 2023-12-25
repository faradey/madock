package migration

import (
	"fmt"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/paths"
	configs2 "github.com/faradey/madock/src/migration/versions/v240/configs"
	"log"
	"os"
)

var versionOptionName string = "madock_version"

func Apply(newAppVersion string) {
	oldAppVersion := ""
	if newAppVersion > "2.4.0" {
		configPath := paths.MakeDirsByPath(paths.GetExecDirPath()+"/projects") + "/config.xml"
		if !paths.IsFileExist(configPath) {
			err := os.WriteFile(configPath, []byte(""), 0755)
			if err != nil {
				log.Fatalln(err)
			}
		}

		config := configs.GetGeneralConfig()
		oldAppVersion = config[versionOptionName]
	} else {
		config := configs2.GetGeneralConfig()
		oldAppVersion = config["MADOCK_VERSION"]
	}

	Execute(oldAppVersion)
	saveNewVersion(newAppVersion)
}

func saveNewVersion(newAppVersion string) {
	fmt.Println(versionOptionName)
	fmt.Println(newAppVersion)
	configs.SetParam(paths.GetExecDirPath()+"/projects/config.xml", versionOptionName, newAppVersion)
}
