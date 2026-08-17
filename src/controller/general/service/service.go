package service

import (
	"strings"

	"github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/logger"
)

var serviceMap = map[string]string{
	"db":                             "db",
	"db2":                            "db2",
	"db/phpmyadmin":                  "phpmyadmin",
	"db2/phpmyadmin":                 "phpmyadmin2",
	"db/pgadmin":                     "pgadmin",
	"db/mongo_express":               "mongo_express",
	"magento/cloud":                  "cloud",
	"magento/mftf":                   "mftf",
	"magento/n98magerun":             "n98magerun",
	"nginx/ssl":                      "ssl",
	"proxy/mailpit":                  "mailpit",
	"nodejs/yarn":                    "yarn",
	"php/ioncube":                    "ioncube",
	"php/xdebug":                     "xdebug",
	"search/elasticsearch":           "elasticsearch",
	"search/elasticsearch/dashboard": "elasticsearch_dashboard",
	"search/opensearch":              "opensearch",
	"search/opensearch/dashboard":    "opensearch_dashboard",
	"search/meilisearch":             "meilisearch",
	"medusa/storefront":              "storefront",
	"saleor/dashboard":               "dashboard",
	"saleor/worker":                  "worker",
	"spree/sidekiq":                  "sidekiq",
	"spree/storefront":               "storefront",
	"sylius/messenger":               "messenger",
	"sylius/encore":                  "encore",
	"artemis":                        "artemis",
	"nodejs/embedded":                "embedded-node",
}

// flagKeys are settings that are themselves the switch, rather than a
// <name>/enabled pair.
//
// Embedded node is the one: it is not a container, it is a runtime added to
// whichever application image the project has, so there is nothing for the word
// "enabled" to belong to. It is listed here rather than shaped into
// nodejs/embedded/enabled because the config is read far more often than a
// service is toggled, and the shorter name is the one people live with.
//
// Until 3.9.8 the same thing was reachable as `service:enable php/nodejs`, and
// that worked by accident: it was in no map, and IsService found it only
// because php/nodejs/enabled happened to exist. Keeping the ergonomics on
// purpose is what this map is for.
var flagKeys = map[string]bool{
	"nodejs/embedded": true,
}

// renamedServices maps a name that no longer exists to the one that replaced
// it.
//
// A customer who has typed `service:enable php/nodejs` for two years should not
// meet "The service doesn't exist." on an upgrade. That message is true and
// useless: it does not say the thing moved, or where to. So the old name keeps
// working, and the command says the new one — which is how somebody learns it,
// rather than by reading a changelog they never opened.
//
// The two entries do not carry the same weight, and it is worth being accurate
// about that. `php/nodejs` was documented in four places and worked for every
// project, because php/nodejs/enabled was in the shipped defaults. `php/yarn`
// was never a registered service at all — in no map, in no defaults — and
// worked only where a platform configurator happened to have written
// php/yarn/enabled: shopify, bigcommerce and sylius. It is aliased for whoever
// scripted it on one of those, not because it was a supported name.
//
// Neither is kept forever. They are cheap while the old name is still in
// people's fingers and in their scripts, and they come out when it is not.
var renamedServices = map[string]string{
	"php/nodejs": "nodejs/embedded",
	"php/yarn":   "nodejs/yarn",
}

// Renamed reports the current name of a service that has been renamed.
func Renamed(name string) (string, bool) {
	current, ok := renamedServices[strings.ToLower(name)]
	return current, ok
}

// IsFlagKey reports whether a config key is itself a service switch, rather
// than the <name>/enabled half of one.
//
// Everything that walks the config looking for services has to ask this, not
// just the one that toggles them: a service you can enable and cannot see in
// the list is worse than one you cannot enable at all.
func IsFlagKey(key string) bool {
	return flagKeys[strings.ToLower(key)]
}

// ConfigKeyOf returns the config key a service name switches.
func ConfigKeyOf(name string) string {
	key := GetByShort(name)
	if flagKeys[key] {
		return key
	}
	return key + "/enabled"
}

// RegisterService adds a service mapping (config key → short name).
func RegisterService(configKey, shortName string) {
	serviceMap[configKey] = shortName
}

func IsService(name string) bool {
	name = strings.ToLower(name)
	configData := configs.GetCurrentProjectConfig()
	name = GetByShort(name)
	if flagKeys[name] {
		if _, ok := configData[name]; ok {
			return true
		}
	}
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
	if current, renamed := Renamed(shortName); renamed {
		return current
	}
	for key, val := range serviceMap {
		if val == shortName {
			shortName = key
			break
		}
	}

	return shortName
}
