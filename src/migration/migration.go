package migration

import (
	"io/ioutil"
	"os"

	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/paths"
)

var versionOptionName string = "MADOCK_VERSION"

func Apply(newAppVersion string) {
	configPath := paths.MakeDirsByPath(paths.GetExecDirPath()+"/projects") + "/config.txt"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		ioutil.WriteFile(configPath, []byte(""), 0755)
	}
	config := configs.GetGeneralConfig()
	oldAppVersion := config[versionOptionName]
	Execute(oldAppVersion)
	saveNewVersion(newAppVersion)
}

func saveNewVersion(newAppVersion string) {
	configs.SetParam(paths.GetExecDirPath()+"/projects/config.txt", versionOptionName, newAppVersion)
}
