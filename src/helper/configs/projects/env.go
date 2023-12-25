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
	config.EnvFile = envFile

	config.Set("PATH", paths.GetRunDirPath())
	config.Set("PLATFORM", defVersions.Platform)
	if projectConf["PLATFORM"] == "magento2" {
		Magento2(config, defVersions, generalConf, projectConf)
	} else if projectConf["PLATFORM"] == "pwa" {
		PWA(config, defVersions, generalConf, projectConf)
	} else if projectConf["PLATFORM"] == "shopify" {
		Shopify(config, defVersions, generalConf, projectConf)
	} else if projectConf["PLATFORM"] == "custom" {
		Custom(config, defVersions, generalConf, projectConf)
	}

	config.Set("CRON_ENABLED", configs2.GetOption("CRON_ENABLED", generalConf, projectConf))

	config.Set("HOSTS", defVersions.Hosts)

	config.Set("SSH_AUTH_TYPE", configs2.GetOption("SSH_AUTH_TYPE", generalConf, projectConf))
	config.Set("SSH_HOST", configs2.GetOption("SSH_HOST", generalConf, projectConf))
	config.Set("SSH_PORT", configs2.GetOption("SSH_PORT", generalConf, projectConf))
	config.Set("SSH_USERNAME", configs2.GetOption("SSH_USERNAME", generalConf, projectConf))
	config.Set("SSH_KEY_PATH", configs2.GetOption("SSH_KEY_PATH", generalConf, projectConf))
	config.Set("SSH_PASSWORD", configs2.GetOption("SSH_PASSWORD", generalConf, projectConf))
	config.Set("SSH_SITE_ROOT_PATH", configs2.GetOption("SSH_SITE_ROOT_PATH", generalConf, projectConf))

	config.Save()
}
