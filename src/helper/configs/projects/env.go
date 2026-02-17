package projects

import (
	configs2 "github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/paths"
	"github.com/faradey/madock/src/model/versions"
)

func SetEnvForProject(projectName string, defVersions versions.ToolsVersions, projectConf map[string]string) {
	generalConf := configs2.GetGeneralConfig()
	config := new(configs2.ConfigLines)
	envFile := paths.MakeDirsByPath(paths.GetExecDirPath()+"/projects/"+projectName) + "/config.xml"
	if paths.IsFileExist(paths.GetRunDirPath() + "/.madock/config.xml") {
		envFile = paths.GetRunDirPath() + "/.madock/config.xml"
	}
	config.EnvFile = envFile
	config.ActiveScope = "default"
	if _, ok := projectConf["activeScope"]; ok {
		config.ActiveScope = projectConf["activeScope"]
	}

	configs2.SetParam(projectName, "path", paths.GetRunDirPath(), "default", configs2.MadockLevelConfigCode)
	/*config.Set("path", paths.GetRunDirPath())*/
	config.Set("platform", defVersions.Platform)
	if defVersions.Language != "" {
		config.Set("language", defVersions.Language)
	}
	platform := defVersions.Platform
	if _, ok := projectConf["platform"]; ok {
		platform = projectConf["platform"]
	}
	if writer, ok := GetEnvWriter(platform); ok {
		writer(config, defVersions, generalConf, projectConf)
	}

	config.Set("cron/enabled", configs2.GetOption("cron/enabled", generalConf, projectConf))

	config.Set("hosts", defVersions.Hosts)

	config.Set("ssh/auth_type", configs2.GetOption("ssh/auth_type", generalConf, projectConf))
	config.Set("ssh/host", configs2.GetOption("ssh/host", generalConf, projectConf))
	config.Set("ssh/port", configs2.GetOption("ssh/port", generalConf, projectConf))
	if !paths.IsFileExist(paths.GetRunDirPath() + "/.madock/config.xml") {
		config.Set("ssh/username", configs2.GetOption("ssh/username", generalConf, projectConf))
		config.Set("ssh/key_path", configs2.GetOption("ssh/key_path", generalConf, projectConf))
		config.Set("ssh/password", configs2.GetOption("ssh/password", generalConf, projectConf))
	}
	config.Set("ssh/site_root_path", configs2.GetOption("ssh/site_root_path", generalConf, projectConf))

	config.Save()
}
