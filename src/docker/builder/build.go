package builder

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/faradey/madock/src/cli/fmtc"
	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/configs/aruntime/nginx"
	"github.com/faradey/madock/src/configs/aruntime/project"
	"github.com/faradey/madock/src/paths"
)

func UpWithBuild() {
	PrepareConfigs()
	DownNginx()
	UpNginxWithBuild()
	upProjectWithBuild()
}

func PrepareConfigs() {
	projectName := paths.GetRunDirName()
	nginx.MakeConf()
	project.MakeConf(projectName)
}

func Down() {
	projectName := paths.GetRunDirName()
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
			"nodetrue",
			"--profile",
			"elasticsearchtrue",
			"--profile",
			"redisdbtrue",
			"--profile",
			"rabbitmqtrue",
			"--profile",
			"kibanatrue",
			"--profile",
			"phpmyadmintrue",
			"down",
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

func Start() {
	projectName := paths.GetRunDirName()
	UpNginx()
	composeFileOS := paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/docker-compose.override.yml"
	profilesOn := []string{
		"compose",
		"-f",
		paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/docker-compose.yml",
		"-f",
		composeFileOS,
		"--profile",
		"nodetrue",
		"--profile",
		"elasticsearchtrue",
		"--profile",
		"redisdbtrue",
		"--profile",
		"rabbitmqtrue",
		"--profile",
		"kibanatrue",
		"--profile",
		"phpmyadmintrue",
		"start",
	}
	cmd := exec.Command("docker", profilesOn...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmtc.ToDoLn("Creating containers")
		UpWithBuild()
	} else {
		projectConfig := configs.GetCurrentProjectConfig()
		if val, ok := projectConfig["CRON_ENABLED"]; ok && val == "true" {
			Cron(true, false)
		} else {
			Cron(false, false)
		}
	}
}

func Stop() {
	projectName := paths.GetRunDirName()
	composeFileOS := paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/docker-compose.override.yml"
	profilesOn := []string{
		"compose",
		"-f",
		paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/docker-compose.yml",
		"-f",
		composeFileOS,
		"--profile",
		"nodetrue",
		"--profile",
		"elasticsearchtrue",
		"--profile",
		"redisdbtrue",
		"--profile",
		"rabbitmqtrue",
		"--profile",
		"kibanatrue",
		"--profile",
		"phpmyadmintrue",
		"stop",
	}
	cmd := exec.Command("docker", profilesOn...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func UpNginx() {
	cmd := exec.Command("docker", "compose", "-f", paths.GetExecDirPath()+"/aruntime/docker-compose.yml", "up", "-d")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		UpNginxWithBuild()
	}
}

func UpNginxWithBuild() {
	PrepareConfigs()
	dockerComposePull([]string{"compose", "-f", paths.GetExecDirPath() + "/aruntime/docker-compose.yml"})
	cmd := exec.Command("docker", "compose", "-f", paths.GetExecDirPath()+"/aruntime/docker-compose.yml", "up", "--build", "--force-recreate", "--no-deps", "-d")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func upProjectWithBuild() {
	projectName := paths.GetRunDirName()
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
		"nodetrue",
		"--profile",
		"elasticsearchtrue",
		"--profile",
		"redisdbtrue",
		"--profile",
		"rabbitmqtrue",
		"--profile",
		"kibanatrue",
		"--profile",
		"phpmyadmintrue",
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
	projectName := paths.GetRunDirName()
	cmd := exec.Command("docker", "exec", "-it", "-u", "www-data", strings.ToLower(projectName)+"-php-1", "bash", "-c", "cd /var/www/html && php bin/magento "+flag)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func Cloud(flag string) {
	projectName := paths.GetRunDirName()
	cmd := exec.Command("docker", "exec", "-it", "-u", "www-data", strings.ToLower(projectName)+"-php-1", "bash", "-c", "cd /var/www/html && magento-cloud "+flag)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func DownloadMagento(edition, version string) {
	projectName := paths.GetRunDirName()
	cmd := exec.Command("docker", "exec", "-it", "-u", "www-data", strings.ToLower(projectName)+"-php-1", "bash", "-c", "cd /var/www/html && mkdir /var/www/html/download-magento && composer create-project --repository-url=https://repo.magento.com/ magento/project-"+edition+"-edition:"+version+" ./download-magento && shopt -s dotglob && mv  -v ./download-magento/* ./ && rmdir ./download-magento && composer install")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func InstallMagento(magentoVer string) {
	projectName := paths.GetRunDirName()
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
		installCommand += "--search-engine=elasticsearch7 " +
			"--elasticsearch-host=elasticsearch " +
			"--elasticsearch-port=9200 " +
			"--elasticsearch-index-prefix=magento2 " +
			"--elasticsearch-timeout=15 " +
			"&& bin/magento module:disable Magento_TwoFactorAuth "
	}
	installCommand += " && bin/magento s:up && bin/magento c:c && bin/magento i:rei && bin/magento c:f"
	cmd := exec.Command("docker", "exec", "-it", "-u", "www-data", strings.ToLower(projectName)+"-php-1", "bash", "-c", "cd /var/www/html && "+installCommand)
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
	projectName := paths.GetRunDirName()
	cmd := exec.Command("docker", "exec", "-it", "-u", "www-data", strings.ToLower(projectName)+"-php-1", "bash", "-c", flag)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func Composer(flag string) {
	projectName := paths.GetRunDirName()
	cmd := exec.Command("docker", "exec", "-it", "-u", "www-data", strings.ToLower(projectName)+"-php-1", "bash", "-c", "cd /var/www/html && composer "+flag)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func Bash(containerName string) {
	projectName := paths.GetRunDirName()
	cmd := exec.Command("docker", "exec", "-it", strings.ToLower(projectName)+"-"+containerName+"-1", "bash")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func CleanCache() {
	projectName := paths.GetRunDirName()
	cmd := exec.Command("docker", "exec", "-it", "-u", "www-data", strings.ToLower(projectName)+"-php-1", "bash", "-c", "cd /var/www/html && rm -f pub/static/deployed_version.txt && rm -Rf pub/static/frontend && rm -Rf pub/static/adminhtml && rm -Rf var/view_preprocessed/pub && rm -Rf generated/code && php bin/magento c:f")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func Node(flag string) {
	projectName := paths.GetRunDirName()
	cmd := exec.Command("docker", "exec", "-it", "-u", "www-data", strings.ToLower(projectName)+"-php-1", "bash", "-c", "cd /var/www/html && "+flag)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func Logs(flag string) {
	projectName := paths.GetRunDirName()
	cmd := exec.Command("docker", "logs", strings.ToLower(projectName)+"-"+flag+"-1")
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
