package versions

import (
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/paths"
)

var mapping = map[string]string{
	"NGINX_UNSECURE_PORT":            "NGINX/UNSECURE/PORT",
	"NGINX_SECURE_PORT":              "NGINX/SECURE/PORT",
	"NGINX_INTERNAL_PORT":            "NGINX/INTERNAL/PORT",
	"PLATFORM":                       "PLATFORM",
	"WORKDIR":                        "WORKDIR",
	"PUBLIC_DIR":                     "PUBLIC_DIR",
	"PHP_VERSION":                    "PHP/VERSION",
	"PHP_COMPOSER_VERSION":           "PHP/COMPOSER/VERSION",
	"PHP_TZ":                         "PHP/TZ",
	"XDEBUG_VERSION":                 "XDEBUG/VERSION",
	"XDEBUG_IDE_KEY":                 "XDEBUG/IDE_KEY",
	"XDEBUG_REMOTE_HOST":             "XDEBUG/REMOTE_HOST",
	"XDEBUG_ENABLED":                 "XDEBUG/ENABLED",
	"XDEBUG_MODE":                    "XDEBUG/MODE",
	"IONCUBE_ENABLED":                "IONCUBE/ENABLED",
	"DB_REPOSITORY":                  "DB/REPOSITORY",
	"DB_ROOT_PASSWORD":               "DB/ROOT_PASSWORD",
	"DB_USER":                        "DB/USER",
	"DB_PASSWORD":                    "DB/PASSWORD",
	"DB_DATABASE":                    "DB/DATABASE",
	"PHPMYADMIN_ENABLED":             "PHPMYADMIN/ENABLED",
	"PHPMYADMIN_REPOSITORY":          "PHPMYADMIN/REPOSITORY",
	"PHPMYADMIN_VERSION":             "PHPMYADMIN/VERSION",
	"DB2_ENABLED":                    "DB2/ENABLED",
	"DB2_REPOSITORY":                 "DB2/REPOSITORY",
	"DB2_ROOT_PASSWORD":              "DB2/ROOT_PASSWORD",
	"DB2_USER":                       "DB2/USER",
	"DB2_PASSWORD":                   "DB2/PASSWORD",
	"DB2_DATABASE":                   "DB2/DATABASE",
	"PHPMYADMIN2_ENABLED":            "PHPMYADMIN2/ENABLED",
	"PHPMYADMIN2_REPOSITORY":         "PHPMYADMIN2/REPOSITORY",
	"PHPMYADMIN2_VERSION":            "PHPMYADMIN2/VERSION",
	"NODEJS_ENABLED":                 "NODEJS/ENABLED",
	"NODEJS_VERSION":                 "NODEJS/VERSION",
	"YARN_ENABLED":                   "YARN/ENABLED",
	"YARN_VERSION":                   "YARN/VERSION",
	"SEARCH_ENGINE":                  "SEARCH_ENGINE",
	"ELASTICSEARCH_ENABLED":          "ELASTICSEARCH/ENABLED",
	"ELASTICSEARCH_REPOSITORY":       "ELASTICSEARCH/REPOSITORY",
	"ELASTICSEARCH_VERSION":          "ELASTICSEARCH/VERSION",
	"KIBANA_ENABLED":                 "KIBANA/ENABLED",
	"KIBANA_REPOSITORY":              "KIBANA/REPOSITORY",
	"OPENSEARCH_ENABLED":             "OPENSEARCH/ENABLED",
	"OPENSEARCH_REPOSITORY":          "OPENSEARCH/REPOSITORY",
	"OPENSEARCH_VERSION":             "OPENSEARCH/VERSION",
	"OPENSEARCHDASHBOARD_ENABLED":    "OPENSEARCHDASHBOARD/ENABLED",
	"OPENSEARCHDASHBOARD_REPOSITORY": "OPENSEARCHDASHBOARD/REPOSITORY",
	"REDIS_ENABLED":                  "REDIS/ENABLED",
	"REDIS_REPOSITORY":               "REDIS/REPOSITORY",
	"RABBITMQ_ENABLED":               "RABBITMQ/ENABLED",
	"RABBITMQ_REPOSITORY":            "RABBITMQ/REPOSITORY",
	"CRON_ENABLED":                   "CRON/ENABLED",
	"SSH_AUTH_TYPE":                  "SSH/AUTH_TYPE",
	"SSH_HOST":                       "SSH/HOST",
	"SSH_PORT":                       "SSH/PORT",
	"SSH_USERNAME":                   "SSH/USERNAME",
	"SSH_KEY_PATH":                   "SSH/KEY_PATH",
	"SSH_PASSWORD":                   "SSH/PASSWORD",
	"SSH_SITE_ROOT_PATH":             "SSH/SITE_ROOT_PATH",
	"HOSTS":                          "HOSTS",
	"SSL":                            "SSL",
	"MAGENTO_RUN_TYPE":               "MAGENTO/RUN_TYPE",
	"MAGENTO_ADMIN_EMAIL":            "MAGENTO/ADMIN_EMAIL",
	"MAGENTO_ADMIN_FIRST_NAME":       "MAGENTO/ADMIN_FIRST_NAME",
	"MAGENTO_ADMIN_LAST_NAME":        "MAGENTO/ADMIN_LAST_NAME",
	"MAGENTO_ADMIN_USER":             "MAGENTO/ADMIN_USER",
	"MAGENTO_ADMIN_PASSWORD":         "MAGENTO/ADMIN_PASSWORD",
	"MAGENTO_ADMIN_FRONTNAME":        "MAGENTO/ADMIN_FRONTNAME",
	"MAGENTO_LOCALE":                 "MAGENTO/LOCALE",
	"MAGENTO_CURRENCY":               "MAGENTO/CURRENCY",
	"MAGENTO_TIMEZONE":               "MAGENTO/TIMEZONE",
	"MFTF_ENABLED":                   "MFTF/ENABLED",
	"MFTF_ADMIN_USER":                "MFTF/ADMIN_USER",
	"MFTF_OTP_SHARED_SECRET":         "MFTF/OTP_SHARED_SECRET",
	"MAGENTOCLOUD_ENABLED":           "MAGENTOCLOUD/ENABLED",
	"MAGENTOCLOUD_USERNAME":          "MAGENTOCLOUD/USERNAME",
	"MAGENTOCLOUD_PASSWORD":          "MAGENTOCLOUD/PASSWORD",
	"MAGENTOCLOUD_PROJECT_NAME":      "MAGENTOCLOUD/PROJECT_NAME",
	"DEFAULT_HOST_FIRST_LEVEL":       "DEFAULT_HOST_FIRST_LEVEL",
	"UBUNTU_VERSION":                 "UBUNTU_VERSION",
	"INTERFACE_IP":                   "INTERFACE_IP",
	"N98MAGERUN_ENABLED":             "N98MAGERUN/ENABLED",
	"PROXY_ENABLED":                  "PROXY/ENABLED",
	"PWA_BACKEND_URL":                "PWA/BACKEND_URL",
	"CONTAINER_NAME_PREFIX":          "CONTAINER_NAME_PREFIX",
}

func V240() {
	execProjectsDirs := paths.GetDirs(paths.GetExecDirPath() + "/projects")
	execPath := paths.GetExecDirPath() + "/projects/"
	projectName := ""
	envFile := ""
	for _, dir := range execProjectsDirs {
		if paths.IsFileExist(execPath + dir + "/env.txt") {
			projectName = dir
			projectConfOnly := configs.GetProjectConfigOnly(projectName)
			projectConf := configs.GetProjectConfig(projectName)
			envFile = paths.MakeDirsByPath(paths.GetExecDirPath()+"/projects/"+projectName) + "/env.txt"
		}
	}
}
