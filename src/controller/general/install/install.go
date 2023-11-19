package install

import (
	"fmt"
	"github.com/faradey/madock/src/cli/fmtc"
	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/versions/magento2"
	"log"
	"os"
	"os/exec"
	"strings"
)

func Execute() {
	projectConf := configs.GetCurrentProjectConfig()
	if projectConf["PLATFORM"] == "magento2" {
		toolsDefVersions := magento2.GetVersions("")
		Magento(configs.GetProjectName(), toolsDefVersions.Magento)
	} else {
		fmtc.Warning("This command is not supported for " + projectConf["PLATFORM"])
	}
}

func Magento(projectName, magentoVer string) {
	projectConf := configs.GetCurrentProjectConfig()
	host := strings.Split(strings.Split(projectConf["HOSTS"], " ")[0], ":")[0]
	installCommand := "bin/magento setup:install " +
		"--base-url=https://" + host + " " +
		"--db-host=db " +
		"--db-name=magento " +
		"--db-user=magento " +
		"--db-password=magento " +
		"--admin-firstname=" + projectConf["MAGENTO_ADMIN_FIRST_NAME"] + " " +
		"--admin-lastname=" + projectConf["MAGENTO_ADMIN_LAST_NAME"] + " " +
		"--admin-email=" + projectConf["MAGENTO_ADMIN_EMAIL"] + " " +
		"--admin-user=" + projectConf["MAGENTO_ADMIN_USER"] + " " +
		"--admin-password=" + projectConf["MAGENTO_ADMIN_PASSWORD"] + " " +
		"--backend-frontname=" + projectConf["MAGENTO_ADMIN_FRONTNAME"] + " " +
		"--language=" + projectConf["MAGENTO_LOCALE"] + " " +
		"--currency=" + projectConf["MAGENTO_CURRENCY"] + " " +
		"--timezone=" + projectConf["MAGENTO_TIMEZONE"] + " " +
		"--use-rewrites=1 "
	if magentoVer >= "2.3.7" {
		searchEngine := projectConf["SEARCH_ENGINE"]
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
	cmd := exec.Command("docker", "exec", "-it", "-u", "www-data", strings.ToLower(projectConf["CONTAINER_NAME_PREFIX"])+strings.ToLower(projectName)+"-php-1", "bash", "-c", "cd "+projectConf["WORKDIR"]+" && "+installCommand)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("")
	fmtc.SuccessLn("[SUCCESS]: Magento installation complete.")
	fmtc.SuccessLn("[SUCCESS]: Magento Admin URI: /" + projectConf["MAGENTO_ADMIN_FRONTNAME"])
	fmtc.SuccessLn("[SUCCESS]: Magento Admin User: " + projectConf["MAGENTO_ADMIN_USER"])
	fmtc.SuccessLn("[SUCCESS]: Magento Admin Password: " + projectConf["MAGENTO_ADMIN_PASSWORD"])
}
