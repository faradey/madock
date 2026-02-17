package service

import (
	"strings"

	"github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/logger"
)

var serviceMap = map[string]string{
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

// RegisterService adds a service mapping (config key → short name).
func RegisterService(configKey, shortName string) {
	serviceMap[configKey] = shortName
}

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
	result := make(map[string]string, len(serviceMap))
	for k, v := range serviceMap {
		result[k] = v
	}
	return result
}

func GetByLong(longName string) string {
	longName = strings.ToLower(longName)
	if val, ok := serviceMap[longName]; ok {
		longName = val
	}

	return longName
}

func GetByShort(shortName string) string {
	shortName = strings.ToLower(shortName)
	for key, val := range serviceMap {
		if val == shortName {
			shortName = key
			break
		}
	}

	return shortName
}
