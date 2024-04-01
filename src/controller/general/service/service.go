package service

import (
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/logger"
	"strings"
)

func IsService(name string) bool {
	name = strings.ToLower(name)
	configData := configs.GetCurrentProjectConfig()
	name = GetByShort(name)
	for key := range configData {
		serviceArr := strings.SplitN(key, "/enabled", 2)
		if serviceArr[0] == name {
			return true
		}
	}

	logger.Fatalln("The service \"" + name + "\" doesn't exist.")

	return false
}

func GetMap() map[string]string {
	return map[string]string{
		"db/phpmyadmin":                  "phpmyadmin",
		"db2/phpmyadmin":                 "phpmyadmin2",
		"magento/cloud":                  "cloud",
		"magento/mftf":                   "mftf",
		"magento/n98magerun":             "n98magerun",
		"nginx/ssl":                      "ssl",
		"nodejs/yarn":                    "yarn",
		"php/ioncube":                    "ioncube",
		"php/xdebug":                     "xdebug",
		"search/elasticsearch":           "elasticsearch",
		"search/elasticsearch/dashboard": "elasticsearch_dashboard",
		"search/opensearch":              "opensearch",
		"search/opensearch/dashboard":    "opensearch_dashboard",
	}
}

func GetByLong(longName string) string {
	mapNames := GetMap()
	longName = strings.ToLower(longName)
	if val, ok := mapNames[longName]; ok {
		longName = val
	}

	return longName
}

func GetByShort(shortName string) string {
	mapNames := GetMap()
	shortName = strings.ToLower(shortName)
	for key, val := range mapNames {
		if val == shortName {
			shortName = key
			break
		}
	}

	return shortName
}
