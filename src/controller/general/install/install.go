package install

import (
	"fmt"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/docker"
	"github.com/faradey/madock/src/helper/logger"
	"github.com/faradey/madock/src/model/versions/magento2"
	"github.com/faradey/madock/src/model/versions/shopware"
	"os"
	"os/exec"
)

func Execute() {
	projectConf := configs.GetCurrentProjectConfig()
	if projectConf["platform"] == "magento2" {
		toolsDefVersions := magento2.GetVersions("")
		Magento(configs.GetProjectName(), toolsDefVersions.PlatformVersion)
	} else if projectConf["platform"] == "shopware" {
		toolsDefVersions := shopware.GetVersions("")
		Shopware(configs.GetProjectName(), toolsDefVersions.PlatformVersion)
	} else {
		fmtc.Warning("This command is not supported for " + projectConf["platform"])
	}
}

func Magento(projectName, magentoVer string) {
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
	if magentoVer >= "2.3.7" {
		searchEngine := projectConf["search/engine"]
		if searchEngine == "Elasticsearch" {
			installCommand += "--search-engine=elasticsearch7 " +
				"--elasticsearch-host=elasticsearch " +
				"--elasticsearch-port=9200 " +
				"--elasticsearch-index-prefix=magento2 " +
				"--elasticsearch-timeout=15 "
		} else if searchEngine == "OpenSearch" {
			if magentoVer >= "2.4.6" {
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

		if magentoVer >= "2.4.6" {
			installCommand += "&& bin/magento module:disable Magento_AdminAdobeImsTwoFactorAuth "
		}
		installCommand += "&& bin/magento module:disable Magento_TwoFactorAuth "
	}
	installCommand += " && bin/magento s:up && bin/magento c:c && bin/magento i:rei | bin/magento c:f"
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
	fmtc.SuccessLn("[SUCCESS]: PlatformVersion installation complete.")
	fmtc.SuccessLn("[SUCCESS]: PlatformVersion Admin URI: /" + projectConf["magento/admin_frontname"])
	fmtc.SuccessLn("[SUCCESS]: PlatformVersion Admin User: " + projectConf["magento/admin_user"])
	fmtc.SuccessLn("[SUCCESS]: PlatformVersion Admin Password: " + projectConf["magento/admin_password"])
}

func Shopware(projectName, magentoVer string) {
	projectConf := configs.GetCurrentProjectConfig()
	host := ""
	hosts := configs.GetHosts(projectConf)
	if len(hosts) > 0 {
		host = hosts[0]["name"]
	}

	/*
		--shop-locale="en-GB" \
		--shop-host="your-shop-url.com" \
		--shop-name="Your Shop Name" \
		--shop-email="admin@yourshop.com" \
		--shop-currency="USD" \
		--admin-username="admin" \
		--admin-password="yourpassword" \
		--admin-email="admin@yourshop.com"
	*/

	installCommand := "bin/console system:install " +
		"--create-database " +
		"--basic-setup " +
		"--database-url=\"mysql://magento:magento@db:3306/magento\" " +
		"--base-url=https://" + host + " " +
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
	searchEngine := projectConf["search/engine"]
	if searchEngine == "Elasticsearch" {
		installCommand += "--search-engine=elasticsearch7 " +
			"--elasticsearch-host=elasticsearch " +
			"--elasticsearch-port=9200 " +
			"--elasticsearch-index-prefix=magento2 " +
			"--elasticsearch-timeout=15 "
	} else if searchEngine == "OpenSearch" {
		installCommand += "--search-engine=opensearch " +
			"--opensearch-host=opensearch " +
			"--opensearch-port=9200 " +
			"--opensearch-index-prefix=magento2 " +
			"--opensearch-timeout=15 "
	}

	installCommand += " && bin/magento s:up && bin/magento c:c && bin/magento i:rei | bin/magento c:f"
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
	fmtc.SuccessLn("[SUCCESS]: PlatformVersion installation complete.")
	fmtc.SuccessLn("[SUCCESS]: PlatformVersion Admin URI: /" + projectConf["magento/admin_frontname"])
	fmtc.SuccessLn("[SUCCESS]: PlatformVersion Admin User: " + projectConf["magento/admin_user"])
	fmtc.SuccessLn("[SUCCESS]: PlatformVersion Admin Password: " + projectConf["magento/admin_password"])
}
