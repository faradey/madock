package install

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/faradey/madock/src/helper/cli/fmtc"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/docker"
	"github.com/faradey/madock/src/helper/logger"
	"github.com/faradey/madock/src/model/versions/magento2"
	"github.com/faradey/madock/src/model/versions/prestashop"
	"github.com/faradey/madock/src/model/versions/shopware"
)

func Execute() {
	projectConf := configs.GetCurrentProjectConfig()
	if projectConf["platform"] == "magento2" {
		toolsDefVersions := magento2.GetVersions("")
		Magento(configs.GetProjectName(), toolsDefVersions.PlatformVersion)
	} else if projectConf["platform"] == "shopware" {
		toolsDefVersions := shopware.GetVersions("")
		Shopware(configs.GetProjectName(), toolsDefVersions.PlatformVersion, false)
	} else if projectConf["platform"] == "prestashop" {
		toolsDefVersions := prestashop.GetVersions("")
		PrestaShop(configs.GetProjectName(), toolsDefVersions.PlatformVersion, false)
	} else {
		fmtc.Warning("This command is not supported for " + projectConf["platform"])
	}
}

func Magento(projectName, platformVer string) {
	projectConf := configs.GetCurrentProjectConfig()
	host := ""
	hosts := configs.GetHosts(projectConf)
	if len(hosts) > 0 {
		host = hosts[0]["name"]
	}
	installCommand := "bin/magento setup:install " +
		"--base-url=https://" + host + " " +
		"--db-host=db " +
		"--db-name=magento " +
		"--db-user=magento " +
		"--db-password=magento " +
		"--admin-firstname=" + projectConf["magento/admin_first_name"] + " " +
		"--admin-lastname=" + projectConf["magento/admin_last_name"] + " " +
		"--admin-email=" + projectConf["magento/admin_email"] + " " +
		"--admin-user=" + projectConf["magento/admin_user"] + " " +
		"--admin-password=" + projectConf["magento/admin_password"] + " " +
		"--backend-frontname=" + projectConf["magento/admin_frontname"] + " " +
		"--language=" + projectConf["magento/locale"] + " " +
		"--currency=" + projectConf["magento/currency"] + " " +
		"--timezone=" + projectConf["magento/timezone"] + " " +
		"--use-rewrites=1 "
	if platformVer >= "2.3.7" {
		searchEngine := projectConf["search/engine"]
		if searchEngine == "Elasticsearch" {
			installCommand += "--search-engine=elasticsearch7 " +
				"--elasticsearch-host=elasticsearch " +
				"--elasticsearch-port=9200 " +
				"--elasticsearch-index-prefix=magento2 " +
				"--elasticsearch-timeout=15 "
		} else if searchEngine == "OpenSearch" {
			if platformVer >= "2.4.6" {
				installCommand += "--search-engine=opensearch " +
					"--opensearch-host=opensearch " +
					"--opensearch-port=9200 " +
					"--opensearch-index-prefix=magento2 " +
					"--opensearch-timeout=15 "
			} else {
				installCommand += "--search-engine=elasticsearch7 " +
					"--elasticsearch-host=opensearch " +
					"--elasticsearch-port=9200 " +
					"--elasticsearch-index-prefix=magento2 " +
					"--elasticsearch-timeout=15 "
			}
		}

		if platformVer >= "2.4.6" {
			installCommand += "&& bin/magento module:disable Magento_AdminAdobeImsTwoFactorAuth "
		}
		installCommand += "&& bin/magento module:disable Magento_TwoFactorAuth "
	}
	installCommand += " && bin/magento setup:upgrade && bin/magento cache:clean && bin/magento indexer:reindex | bin/magento cache:flush"
	fmt.Println(installCommand)
	cmd := exec.Command("docker", "exec", "-it", "-u", "www-data", docker.GetContainerName(projectConf, projectName, "php"), "bash", "-c", "cd "+projectConf["workdir"]+" && "+installCommand)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		logger.Fatal(err)
	}
	fmt.Println("")
	fmtc.SuccessLn("[SUCCESS]: Magento installation complete.")
	fmtc.SuccessLn("[SUCCESS]: Magento Admin URI: /" + projectConf["magento/admin_frontname"])
	fmtc.SuccessLn("[SUCCESS]: Magento Admin User: " + projectConf["magento/admin_user"])
	fmtc.SuccessLn("[SUCCESS]: Magento Admin Password: " + projectConf["magento/admin_password"])
}

func Shopware(projectName, platformVer string, isSampleData bool) {
	projectConf := configs.GetCurrentProjectConfig()
	host := ""
	hosts := configs.GetHosts(projectConf)
	if len(hosts) > 0 {
		host = hosts[0]["name"]
	}

	installCommand := "sed -i 's/APP_URL=http:\\/\\/127.0.0.1:8000/APP_URL=https:\\/\\/" + host + "/g' .env "
	installCommand += "&& sed -i 's/DATABASE_URL=mysql:\\/\\/root:root@localhost\\/shopware/DATABASE_URL=mysql:\\/\\/magento:magento@db:3306\\/magento/g' .env "
	searchEngine := projectConf["search/engine"]
	if searchEngine == "Elasticsearch" {
		installCommand += "&& sed -i 's/SHOPWARE_ES_ENABLED=0/SHOPWARE_ES_ENABLED=1/g' .env "
		installCommand += "&& sed -i 's/OPENSEARCH_URL=http:\\/\\/localhost:9200/OPENSEARCH_URL=http:\\/\\/elasticsearch:9200/g' .env "
		installCommand += "&& sed -i 's/SHOPWARE_ES_INDEXING_ENABLED=0/SHOPWARE_ES_INDEXING_ENABLED=1/g' .env "
		installCommand += "&& sed -i 's/SHOPWARE_ES_INDEX_PREFIX=sw/SHOPWARE_ES_INDEX_PREFIX=swlocal/g' .env "
	} else if searchEngine == "OpenSearch" {
		installCommand += "&& sed -i 's/SHOPWARE_ES_ENABLED=0/SHOPWARE_ES_ENABLED=1/g' .env "
		installCommand += "&& sed -i 's/OPENSEARCH_URL=http:\\/\\/localhost:9200/OPENSEARCH_URL=opensearch:9200/g' .env "
		installCommand += "&& sed -i 's/SHOPWARE_ES_INDEXING_ENABLED=0/SHOPWARE_ES_INDEXING_ENABLED=1/g' .env "
		installCommand += "&& sed -i 's/SHOPWARE_ES_INDEX_PREFIX=sw/SHOPWARE_ES_INDEX_PREFIX=swlocal/g' .env "
	}

	// replace SHOPWARE_HTTP_CACHE_ENABLED=1 to SHOPWARE_HTTP_CACHE_ENABLED=0
	installCommand += "&& sed -i 's/SHOPWARE_HTTP_CACHE_ENABLED=1/SHOPWARE_HTTP_CACHE_ENABLED=0/g' .env "
	installCommand += "&& sed -i 's/STOREFRONT_PROXY_URL=http:\\/\\/localhost/STOREFRONT_PROXY_URL=https:\\/\\/" + host + "/g' .env "
	installCommand += "&& bin/console system:setup "
	installCommand += "&& bin/console system:install " +
		"--basic-setup " +
		"--shop-name=\"Your Shop Name\" " +
		"--shop-email=\"" + projectConf["magento/admin_email"] + "\" " +
		"--shop-locale=\"en-GB\" " +
		"--shop-currency=\"USD\" " +
		"&& composer update "

	if isSampleData {
		installCommand += "&& composer require swag/demo-data shopware/dev-tools && bin/console framework:demodata "
	}
	installCommand += "&& bin/console es:index "

	fmt.Println(installCommand)
	cmd := exec.Command("docker", "exec", "-it", "-u", "www-data", docker.GetContainerName(projectConf, projectName, "php"), "bash", "-c", "cd "+projectConf["workdir"]+" && "+installCommand)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		logger.Fatal(err)
	}
	fmt.Println("")
	fmtc.SuccessLn("[SUCCESS]: Shopware installation complete.")
	fmtc.SuccessLn("[SUCCESS]: Shopware Admin URI: /admin")
	fmtc.SuccessLn("[SUCCESS]: Shopware Admin User: admin")
	fmtc.SuccessLn("[SUCCESS]: Shopware Admin Password: shopware")
}

func PrestaShop(projectName, platformVer string, isSampleData bool) {
	projectConf := configs.GetCurrentProjectConfig()
	host := ""
	hosts := configs.GetHosts(projectConf)
	if len(hosts) > 0 {
		host = hosts[0]["name"]
	}

	installCommand := "php install/index_cli.php " +
		"--domain=" + host + " " +
		"--db_server=db " +
		"--db_name=magento " +
		"--db_user=magento " +
		"--db_password=magento " +
		"--firstname=" + projectConf["magento/admin_first_name"] + " " +
		"--lastname=" + projectConf["magento/admin_last_name"] + " " +
		"--email=" + projectConf["magento/admin_email"] + " " +
		"--password=" + projectConf["magento/admin_password"] + " " +
		"--timezone=" + projectConf["magento/timezone"] + " " +
		"--rewrite=1 " + " " +
		"--ssl=1 "

	if isSampleData {
		installCommand += " --fixtures=1 "
	}

	fmt.Println(installCommand)
	cmd := exec.Command("docker", "exec", "-it", "-u", "www-data", docker.GetContainerName(projectConf, projectName, "php"), "bash", "-c", "cd "+projectConf["workdir"]+" && "+installCommand)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		logger.Fatal(err)
	}
	fmt.Println("")
	fmtc.SuccessLn("[SUCCESS]: PrestaShop installation complete.")
	fmtc.SuccessLn("[SUCCESS]: PrestaShop Admin URI: /admin")
	fmtc.SuccessLn("[SUCCESS]: PrestaShop Admin User: " + projectConf["magento/admin_email"])
	fmtc.SuccessLn("[SUCCESS]: PrestaShop Admin Password: " + projectConf["magento/admin_password"])
}
