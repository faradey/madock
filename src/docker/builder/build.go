package builder

import (
	"fmt"
	"github.com/faradey/madock/src/cli/attr"
	"github.com/faradey/madock/src/cli/fmtc"
	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/configs/aruntime/nginx"
	"github.com/faradey/madock/src/configs/aruntime/project"
	"github.com/faradey/madock/src/helper"
	"github.com/faradey/madock/src/paths"
	"github.com/gosimple/hashdir"
	"log"
	"os"
	"os/exec"
	"os/user"
	"strings"
)

func UpWithBuild() {
	DownNginx()
	UpNginx()
	upProjectWithBuild(attr.Options.WithChown)
}

func Down(withVolumes bool) {
	projectName := configs.GetProjectName()
	composeFile := paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/docker-compose.yml"
	composeFileOS := paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/docker-compose.override.yml"
	if _, err := os.Stat(composeFile); !os.IsNotExist(err) {
		profilesOn := []string{
			"compose",
			"-f",
			composeFile,
			"-f",
			composeFileOS,
			"--profile",
			"elasticsearchtrue",
			"--profile",
			"opensearchtrue",
			"--profile",
			"redisdbtrue",
			"--profile",
			"rabbitmqtrue",
			"--profile",
			"kibanatrue",
			"--profile",
			"opensearchdashboardtrue",
			"--profile",
			"phpmyadmintrue",
			"--profile",
			"db2true",
			"--profile",
			"phpmyadmin2true",
			"--profile",
			"xdebugtrue",
		}

		profilesOn = append(profilesOn, "down")

		if withVolumes {
			profilesOn = append(profilesOn, "-v")
			profilesOn = append(profilesOn, "--rmi")
			profilesOn = append(profilesOn, "all")
		}

		cmd := exec.Command("docker", profilesOn...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			fmt.Println(err)
		}
	}
}

func UpNginx() {
	UpNginxWithBuild()
}

func UpNginxWithBuild() {
	projectName := configs.GetProjectName()
	nginx.MakeConf()
	project.MakeConf(projectName)
	projectConf := configs.GetCurrentProjectConfig()
	dirHash, err := hashdir.Make(paths.GetExecDirPath()+"/aruntime/projects/"+projectName+"/ctx", "md5")
	dockerComposeHash, err := helper.HashFile(paths.GetExecDirPath()+"/aruntime/projects/"+projectName+"/docker-compose.yml", "md5")
	dockerComposeOverHash, err := helper.HashFile(paths.GetExecDirPath()+"/aruntime/projects/"+projectName+"/docker-compose.override.yml", "md5")
	dirHash = dirHash + dockerComposeHash + dockerComposeOverHash
	doNeedRunAruntime := true
	if _, err := os.Stat(paths.GetExecDirPath() + "/aruntime/docker-compose.yml"); !os.IsNotExist(err) {
		cmd := exec.Command("docker", "compose", "-f", paths.GetExecDirPath()+"/aruntime/docker-compose.yml", "ps", "--format", "json")
		result, err := cmd.CombinedOutput()
		if err != nil {
			log.Fatal(err)
		}
		if len(result) > 100 {
			doNeedRunAruntime = false
		}
	}
	if (err != nil || dirHash != projectConf["CACHE_HASH"] || doNeedRunAruntime) && projectConf["PROXY_ENABLED"] == "true" {
		ctxPath := paths.MakeDirsByPath(paths.GetExecDirPath() + "/aruntime/ctx")
		nginx.GenerateSslCert(ctxPath, false)
		envFile := paths.MakeDirsByPath(paths.GetExecDirPath()+"/projects/"+projectName) + "/env.txt"
		configs.SetParam(envFile, "CACHE_HASH", dirHash)
		dockerComposePull([]string{"compose", "-f", paths.GetExecDirPath() + "/aruntime/docker-compose.yml"})
		cmd := exec.Command("docker", "compose", "-f", paths.GetExecDirPath()+"/aruntime/docker-compose.yml", "up", "--build", "--force-recreate", "--no-deps", "-d")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			log.Fatal(err)
		}
	}
}

func upProjectWithBuild(withChown bool) {
	projectName := configs.GetProjectName()
	if _, err := os.Stat(paths.GetExecDirPath() + "/aruntime/.composer"); os.IsNotExist(err) {
		err = os.Chmod(paths.MakeDirsByPath(paths.GetExecDirPath()+"/aruntime/.composer"), 0777)
		if err != nil {
			log.Fatal(err)
		}
	}

	composerGlobalDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	} else {
		if _, err := os.Stat(composerGlobalDir + "/.composer"); os.IsNotExist(err) {
			paths.MakeDirsByPath(composerGlobalDir + "/.composer")
		}
	}

	src := paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/composer"

	if fi, err := os.Lstat(src); err == nil {
		if fi.Mode()&os.ModeSymlink != os.ModeSymlink {
			err := os.RemoveAll(src)
			if err == nil {
				err := os.Symlink(composerGlobalDir+"/.composer", src)
				if err != nil {
					log.Fatal(err)
				}
			} else {
				fmt.Println(err)
			}
		}
	} else {
		err := os.Symlink(composerGlobalDir+"/.composer", src)
		if err != nil {
			log.Fatal(err)
		}
	}

	sshDir := paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/ssh"

	if fi, err := os.Lstat(sshDir); err == nil {
		if fi.Mode()&os.ModeSymlink != os.ModeSymlink {
			err := os.RemoveAll(sshDir)
			if err == nil {
				err := os.Symlink(composerGlobalDir+"/.ssh", sshDir)
				if err != nil {
					log.Fatal(err)
				}
			} else {
				fmt.Println(err)
			}
		}
	} else {
		err := os.Symlink(composerGlobalDir+"/.ssh", sshDir)
		if err != nil {
			log.Fatal(err)
		}
	}

	composeFileOS := paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/docker-compose.override.yml"
	profilesOn := []string{
		"compose",
		"-f",
		paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/docker-compose.yml",
		"-f",
		composeFileOS,
		"--profile",
		"elasticsearchtrue",
		"--profile",
		"opensearchtrue",
		"--profile",
		"redisdbtrue",
		"--profile",
		"rabbitmqtrue",
		"--profile",
		"kibanatrue",
		"--profile",
		"opensearchdashboardtrue",
		"--profile",
		"phpmyadmintrue",
		"--profile",
		"db2true",
		"--profile",
		"phpmyadmin2true",
		"--profile",
		"xdebugtrue",
		"up",
		"--build",
		"--force-recreate",
		"--no-deps",
		"-d",
	}
	dockerComposePull([]string{"compose", "-f",
		paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/docker-compose.yml",
		"-f",
		composeFileOS})
	cmd := exec.Command("docker", profilesOn...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	projectConfig := configs.GetCurrentProjectConfig()

	if val, ok := projectConfig["CRON_ENABLED"]; ok && val == "true" {
		Cron(true, false)
	} else {
		Cron(false, false)
	}

	if withChown {
		usr, _ := user.Current()
		cmd := exec.Command("docker", "exec", "-it", "-u", "root", strings.ToLower(projectConfig["CONTAINER_NAME_PREFIX"])+strings.ToLower(projectName)+"-php-1", "bash", "-c", "chown -R "+usr.Uid+":"+usr.Gid+" "+projectConfig["WORKDIR"]+" && chown -R "+usr.Uid+":"+usr.Gid+" /var/www/.composer")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			log.Fatal(err)
		}
	}
}

func dockerComposePull(composeFiles []string) {
	composeFiles = append(composeFiles, "pull")
	cmd := exec.Command("docker", composeFiles...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

func DownNginx() {
	composeFile := paths.GetExecDirPath() + "/aruntime/docker-compose.yml"
	if _, err := os.Stat(composeFile); !os.IsNotExist(err) {
		cmd := exec.Command("docker", "compose", "-f", composeFile, "down")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			fmt.Println(err)
		}
	}
}

func StopNginx() {
	composeFile := paths.GetExecDirPath() + "/aruntime/docker-compose.yml"
	if _, err := os.Stat(composeFile); !os.IsNotExist(err) {
		cmd := exec.Command("docker", "compose", "-f", composeFile, "stop")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			fmt.Println(err)
		}
	}
}

func Magento(flag string) {
	projectName := configs.GetProjectName()
	projectConfig := configs.GetCurrentProjectConfig()
	cmd := exec.Command("docker", "exec", "-it", "-u", "www-data", strings.ToLower(projectConfig["CONTAINER_NAME_PREFIX"])+strings.ToLower(projectName)+"-php-1", "bash", "-c", "cd "+projectConfig["WORKDIR"]+" && php bin/magento "+flag)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func Mftf(flag string) {
	projectName := configs.GetProjectName()
	projectConfig := configs.GetCurrentProjectConfig()
	cmd := exec.Command("docker", "exec", "-it", "-u", "www-data", strings.ToLower(projectConfig["CONTAINER_NAME_PREFIX"])+strings.ToLower(projectName)+"-php-1", "bash", "-c", "cd "+projectConfig["WORKDIR"]+" && php vendor/bin/mftf "+flag)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func PWA(flag string) {
	projectName := configs.GetProjectName()
	projectConfig := configs.GetCurrentProjectConfig()
	cmd := exec.Command("docker", "exec", "-it", "-u", "www-data", strings.ToLower(projectConfig["CONTAINER_NAME_PREFIX"])+strings.ToLower(projectName)+"-nodejs-1", "bash", "-c", "cd "+projectConfig["WORKDIR"]+" && "+flag)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func Cloud(flag string) {
	projectName := configs.GetProjectName()
	projectConfig := configs.GetCurrentProjectConfig()
	cmd := exec.Command("docker", "exec", "-it", "-u", "www-data", strings.ToLower(projectConfig["CONTAINER_NAME_PREFIX"])+strings.ToLower(projectName)+"-php-1", "bash", "-c", "cd "+projectConfig["WORKDIR"]+" && magento-cloud "+flag)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func DownloadMagento(projectName, edition, version string) {
	projectConfig := configs.GetCurrentProjectConfig()
	sampleData := ""
	if attr.Options.SampleData {
		sampleData = " && bin/magento sampledata:deploy"
	}
	command := []string{
		"exec",
		"-it",
		"-u",
		"www-data",
		strings.ToLower(projectConfig["CONTAINER_NAME_PREFIX"]) + strings.ToLower(projectName) + "-php-1",
		"bash",
		"-c",
		"cd " + projectConfig["WORKDIR"] + " " +
			"&& rm -r -f " + projectConfig["WORKDIR"] + "/download-magento123456789 " +
			"&& mkdir " + projectConfig["WORKDIR"] + "/download-magento123456789 " +
			"&& composer create-project --repository-url=https://repo.magento.com/ magento/project-" + edition + "-edition:" + version + " ./download-magento123456789 " +
			"&& shopt -s dotglob " +
			"&& mv  -v ./download-magento123456789/* ./ " +
			"&& rm -r -f ./download-magento123456789 " +
			"&& composer install" + sampleData,
	}
	cmd := exec.Command("docker", command...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func InstallMagento(projectName, magentoVer string) {
	projectConfig := configs.GetCurrentProjectConfig()
	host := strings.Split(strings.Split(projectConfig["HOSTS"], " ")[0], ":")[0]
	installCommand := "bin/magento setup:install " +
		"--base-url=https://" + host + " " +
		"--db-host=db " +
		"--db-name=magento " +
		"--db-user=magento " +
		"--db-password=magento " +
		"--admin-firstname=" + projectConfig["MAGENTO_ADMIN_FIRST_NAME"] + " " +
		"--admin-lastname=" + projectConfig["MAGENTO_ADMIN_LAST_NAME"] + " " +
		"--admin-email=" + projectConfig["MAGENTO_ADMIN_EMAIL"] + " " +
		"--admin-user=" + projectConfig["MAGENTO_ADMIN_USER"] + " " +
		"--admin-password=" + projectConfig["MAGENTO_ADMIN_PASSWORD"] + " " +
		"--backend-frontname=" + projectConfig["MAGENTO_ADMIN_FRONTNAME"] + " " +
		"--language=" + projectConfig["MAGENTO_LOCALE"] + " " +
		"--currency=" + projectConfig["MAGENTO_CURRENCY"] + " " +
		"--timezone=" + projectConfig["MAGENTO_TIMEZONE"] + " " +
		"--use-rewrites=1 "
	if magentoVer >= "2.3.7" {
		searchEngine := projectConfig["SEARCH_ENGINE"]
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
	cmd := exec.Command("docker", "exec", "-it", "-u", "www-data", strings.ToLower(projectConfig["CONTAINER_NAME_PREFIX"])+strings.ToLower(projectName)+"-php-1", "bash", "-c", "cd "+projectConfig["WORKDIR"]+" && "+installCommand)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("")
	fmtc.SuccessLn("[SUCCESS]: Magento installation complete.")
	fmtc.SuccessLn("[SUCCESS]: Magento Admin URI: /" + projectConfig["MAGENTO_ADMIN_FRONTNAME"])
	fmtc.SuccessLn("[SUCCESS]: Magento Admin User: " + projectConfig["MAGENTO_ADMIN_USER"])
	fmtc.SuccessLn("[SUCCESS]: Magento Admin Password: " + projectConfig["MAGENTO_ADMIN_PASSWORD"])
}

func Cli(flag string) {
	projectName := configs.GetProjectName()
	// get project config
	projectConfig := configs.GetCurrentProjectConfig()
	containerSlug := "php"
	if projectConfig["PLATFORM"] == "pwa" {
		containerSlug = "nodejs"
	}
	cmd := exec.Command("docker", "exec", "-it", "-u", "www-data", strings.ToLower(projectConfig["CONTAINER_NAME_PREFIX"])+strings.ToLower(projectName)+"-"+containerSlug+"-1", "bash", "-c", flag)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func Composer(flag string) {
	projectName := configs.GetProjectName()
	projectConfig := configs.GetCurrentProjectConfig()
	cmd := exec.Command("docker", "exec", "-it", "-u", "www-data", strings.ToLower(projectConfig["CONTAINER_NAME_PREFIX"])+strings.ToLower(projectName)+"-php-1", "bash", "-c", "cd "+projectConfig["WORKDIR"]+" && composer "+flag)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func CleanCache() {
	projectName := configs.GetProjectName()
	projectConfig := configs.GetCurrentProjectConfig()
	cmd := exec.Command("docker", "exec", "-it", "-u", "www-data", strings.ToLower(projectConfig["CONTAINER_NAME_PREFIX"])+strings.ToLower(projectName)+"-php-1", "bash", "-c", "cd "+projectConfig["WORKDIR"]+" && rm -f pub/static/deployed_version.txt && rm -Rf pub/static/frontend && rm -Rf pub/static/adminhtml && rm -Rf var/view_preprocessed/pub && rm -Rf generated/code && php bin/magento c:f")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func N98(flag string) {
	projectName := configs.GetProjectName()
	projectConfig := configs.GetCurrentProjectConfig()
	cmd := exec.Command("docker", "exec", "-it", "-u", "www-data", strings.ToLower(projectConfig["CONTAINER_NAME_PREFIX"])+strings.ToLower(projectName)+"-php-1", "bash", "-c", "cd "+projectConfig["WORKDIR"]+" && /var/www/n98magerun/n98-magerun2.phar "+flag)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func Node(flag string) {
	projectName := configs.GetProjectName()
	projectConfig := configs.GetCurrentProjectConfig()
	cmd := exec.Command("docker", "exec", "-it", "-u", "www-data", strings.ToLower(projectConfig["CONTAINER_NAME_PREFIX"])+strings.ToLower(projectName)+"-php-1", "bash", "-c", "cd "+projectConfig["WORKDIR"]+" && "+flag)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func Logs(flag string) {
	projectName := configs.GetProjectName()
	projectConfig := configs.GetCurrentProjectConfig()
	cmd := exec.Command("docker", "logs", strings.ToLower(projectConfig["CONTAINER_NAME_PREFIX"])+strings.ToLower(projectName)+"-"+flag+"-1")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func SslRebuild() {
	ctxPath := paths.MakeDirsByPath(paths.GetExecDirPath() + "/aruntime/ctx")
	nginx.GenerateSslCert(ctxPath, true)
}

func Shopify(flag string) {
	projectName := configs.GetProjectName()
	projectConfig := configs.GetCurrentProjectConfig()
	cmd := exec.Command("docker", "exec", "-it", "-u", "www-data", strings.ToLower(projectConfig["CONTAINER_NAME_PREFIX"])+strings.ToLower(projectName)+"-php-1", "bash", "-c", "cd "+projectConfig["WORKDIR"]+" && "+flag)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func ShopifyWeb(flag string) {
	projectName := configs.GetProjectName()
	projectConfig := configs.GetCurrentProjectConfig()
	cmd := exec.Command("docker", "exec", "-it", "-u", "www-data", strings.ToLower(projectConfig["CONTAINER_NAME_PREFIX"])+strings.ToLower(projectName)+"-php-1", "bash", "-c", "cd "+projectConfig["WORKDIR"]+"/web && "+flag)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func ShopifyWebFrontend(flag string) {
	projectName := configs.GetProjectName()
	projectConfig := configs.GetCurrentProjectConfig()
	cmd := exec.Command("docker", "exec", "-it", "-u", "www-data", strings.ToLower(projectConfig["CONTAINER_NAME_PREFIX"])+strings.ToLower(projectName)+"-php-1", "bash", "-c", "cd "+projectConfig["WORKDIR"]+"/web/frontend && "+flag)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}
