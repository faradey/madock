package migration

import (
	configs2 "github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/paths"
	"os"
)

var versionOptionName string = "MADOCK_VERSION"

func Apply(newAppVersion string) {
	configPath := paths.MakeDirsByPath(paths.GetExecDirPath()+"/projects") + "/config.txt"
	if !paths.IsFileExist(configPath) {
		os.WriteFile(configPath, []byte(""), 0755)
	}
	config := configs2.GetGeneralConfig()
	oldAppVersion := config[versionOptionName]
	Execute(oldAppVersion)
	saveNewVersion(newAppVersion)
}

func saveNewVersion(newAppVersion string) {
	configs2.SetParam(paths.GetExecDirPath()+"/projects/config.txt", versionOptionName, newAppVersion)
}
