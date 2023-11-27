package versions

import (
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/paths"
	"os"
	"strings"
)

func V140() {
	mapNames := map[string]string{
		"PHP_MODULE_XDEBUG":      "XDEBUG_ENABLED",
		"PHP_MODULE_IONCUBE":     "IONCUBE_ENABLED",
		"PHPMYADMIN_ENABLE":      "PHPMYADMIN_ENABLED",
		"NODEJS_ENABLE":          "NODEJS_ENABLED",
		"ELASTICSEARCH_ENABLE":   "ELASTICSEARCH_ENABLED",
		"KIBANA_ENABLE":          "KIBANA_ENABLED",
		"REDIS_ENABLE":           "REDIS_ENABLED",
		"RABBITMQ_ENABLE":        "RABBITMQ_ENABLED",
		"PHP_XDEBUG_VERSION":     "XDEBUG_VERSION",
		"PHP_XDEBUG_IDE_KEY":     "XDEBUG_IDE_KEY",
		"PHP_XDEBUG_REMOTE_HOST": "XDEBUG_REMOTE_HOST",
	}
	configs.ChangeParamName(paths.GetExecDirPath()+"/config.txt", mapNames)
	configs.ChangeParamName(paths.GetExecDirPath()+"/projects/config.txt", mapNames)
	projectsPath := paths.GetExecDirPath() + "/projects"
	dirs := paths.GetDirs(projectsPath)
	for _, val := range dirs {
		configs.ChangeParamName(projectsPath+"/"+val+"/env.txt", mapNames)
		dockerFiles := paths.GetFilesRecursively(projectsPath + "/" + val + "/docker")
		if len(dockerFiles) > 0 {
			for _, pth := range dockerFiles {
				b, err := os.ReadFile(pth)
				if err == nil {
					str := string(b)
					for from, to := range mapNames {
						str = strings.Replace(str, "{{{"+from+"}}}", "{{{"+to+"}}}", -1)
					}
					os.WriteFile(pth, []byte(str), 0755)
				}
			}
		}
	}
}
