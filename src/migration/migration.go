package migration

import (
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/logger"
	"github.com/faradey/madock/src/helper/paths"
	configs2 "github.com/faradey/madock/src/migration/versions/v240/configs"
	"os"
)

var versionOptionName string = "madock_version"

func Apply(newAppVersion string) {
	oldAppVersion := ""
	oldAppVersionXml := ""
	oldAppVersionTxt := ""

	configPath := paths.MakeDirsByPath(paths.GetExecDirPath()+"/projects") + "/config.xml"
	if !paths.IsFileExist(configPath) {
		paths.MakeDirsByPath(paths.CacheDir())
		err := os.WriteFile(configPath, []byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<config>\n<scopes>\n<default></default>\n</scopes>\n</config>"), 0755)
		if err != nil {
			logger.Fatalln(err)
		}
	} else {
		config := configs.GetGeneralConfig()
		oldAppVersionXml = config[versionOptionName]
	}

	if paths.IsFileExist(paths.GetExecDirPath() + "/projects/config.txt") {
		config := configs2.GetGeneralConfig()
		oldAppVersionTxt = config["MADOCK_VERSION"]
		if oldAppVersionTxt <= "2.4.0" {
			configs2.SetParam(paths.GetExecDirPath()+"/projects/config.txt", "MADOCK_VERSION", newAppVersion)
		}
	}

	if oldAppVersionXml > oldAppVersionTxt {
		oldAppVersion = oldAppVersionXml
	} else {
		oldAppVersion = oldAppVersionTxt
	}

	if oldAppVersion < newAppVersion {
		Execute(oldAppVersion)
		saveNewVersion(newAppVersion)
	}
}

func saveNewVersion(newAppVersion string) {
	configs.SetParam(configs.MainConfigCode, versionOptionName, newAppVersion, "default", configs.MainConfigCode)
}
