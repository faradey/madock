package migration

import (
	"github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/logger"
	"github.com/faradey/madock/v3/src/helper/paths"
	configs2 "github.com/faradey/madock/v3/src/migration/versions/v240/configs"
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

	// Compared as versions, not as strings. "3.9.8" < "3.9.10" is **false** as
	// a string — '8' sorts after '1' — so an installation on 3.9.8 upgrading to
	// 3.9.10 would have run no migrations at all, and the one that renames
	// php/nodejs/enabled would simply never have happened for anybody. Silent
	// in both directions: nothing fails, the key is just gone.
	if olderThan(oldAppVersion, newAppVersion) {
		Execute(oldAppVersion)
		saveNewVersion(newAppVersion)
	}
}

func saveNewVersion(newAppVersion string) {
	configs.SetParam(configs.MainConfigCode, versionOptionName, newAppVersion, "default", configs.MainConfigCode)
}
