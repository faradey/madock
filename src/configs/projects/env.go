package projects

import (
	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/paths"
	"github.com/faradey/madock/src/versions"
)

func SetEnvForProject(projectName string, defVersions versions.ToolsVersions, projectConfig map[string]string) {
	generalConf := configs.GetGeneralConfig()
	config := new(configs.ConfigLines)
	envFile := paths.MakeDirsByPath(paths.GetExecDirPath()+"/projects/"+projectName) + "/env.txt"
	config.EnvFile = envFile
	if len(projectConfig) > 0 {
		config.IsEnv = true
	}

	config.AddOrSetLine("PATH", paths.GetRunDirPath())
	config.AddOrSetLine("PLATFORM", defVersions.Platform)
	if projectConfig["PLATFORM"] == "magento2" {
		Magento2(config, defVersions, generalConf, projectConfig)
	} else if projectConfig["PLATFORM"] == "pwa" {
		PWA(config, defVersions, generalConf, projectConfig)
	} else if projectConfig["PLATFORM"] == "shopify" {
		Shopify(config, defVersions, generalConf, projectConfig)
	}

	if !config.IsEnv {
		config.AddEmptyLine()
	}

	config.AddOrSetLine("CRON_ENABLED", configs.GetOption("CRON_ENABLED", generalConf, projectConfig))

	if !config.IsEnv {
		config.AddEmptyLine()
	}

	config.AddOrSetLine("HOSTS", defVersions.Hosts)

	if !config.IsEnv {
		config.AddEmptyLine()
	}

	config.AddOrSetLine("SSH_AUTH_TYPE", configs.GetOption("SSH_AUTH_TYPE", generalConf, projectConfig))
	config.AddOrSetLine("SSH_HOST", configs.GetOption("SSH_HOST", generalConf, projectConfig))
	config.AddOrSetLine("SSH_PORT", configs.GetOption("SSH_PORT", generalConf, projectConfig))
	config.AddOrSetLine("SSH_USERNAME", configs.GetOption("SSH_USERNAME", generalConf, projectConfig))
	config.AddOrSetLine("SSH_KEY_PATH", configs.GetOption("SSH_KEY_PATH", generalConf, projectConfig))
	config.AddOrSetLine("SSH_PASSWORD", configs.GetOption("SSH_PASSWORD", generalConf, projectConfig))
	config.AddOrSetLine("SSH_SITE_ROOT_PATH", configs.GetOption("SSH_SITE_ROOT_PATH", generalConf, projectConfig))

	if !config.IsEnv {
		config.SaveLines()
	}
}
