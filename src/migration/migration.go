package migration

import (
	"github.com/faradey/madock/src/helper/paths"
	"os"

	"github.com/faradey/madock/src/configs"
)

var versionOptionName string = "MADOCK_VERSION"

func Apply(newAppVersion string) {
	configPath := paths.MakeDirsByPath(paths.GetExecDirPath()+"/projects") + "/config.txt"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		os.WriteFile(configPath, []byte(""), 0755)
	}
	config := configs.GetGeneralConfig()
	oldAppVersion := config[versionOptionName]
	Execute(oldAppVersion)
	saveNewVersion(newAppVersion)
}

func saveNewVersion(newAppVersion string) {
	configs.SetParam(paths.GetExecDirPath()+"/projects/config.txt", versionOptionName, newAppVersion)
}
