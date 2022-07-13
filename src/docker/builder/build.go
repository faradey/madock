package builder

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"
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
	projectConf := configs.GetCurrentProjectConfig()
	if runtime.GOOS == "darwin" && projectConf["MUTAGEN_USE"] != "false" {
		clearMutagen(projectName, "php")
	}
	composeFile := paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/docker-compose.yml"
	composeFileOS := paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/docker-compose.override.yml"
	if _, err := os.Stat(composeFile); !os.IsNotExist(err) {
		profilesOn := []string{
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
		cmd := exec.Command("docker-compose", profilesOn...)
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
	PrepareConfigs()
	UpNginx()
	composeFileOS := paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/docker-compose.override.yml"
	profilesOn := []string{
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
	cmd := exec.Command("docker-compose", profilesOn...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmtc.ToDoLn("Creating containers")
		UpWithBuild()
	} else {
		projectConfig := configs.GetCurrentProjectConfig()
		if val, ok := projectConfig["CRON_ENABLED"]; ok && val == "true" {
			Cron("on", false)
		} else {
			Cron("off", false)
		}

		if runtime.GOOS == "darwin" && projectConfig["MUTAGEN_USE"] != "false" {
			syncMutagen(projectName, "php", "www-data")
		}
	}
}

func Stop() {
	projectName := paths.GetRunDirName()
	projectConf := configs.GetCurrentProjectConfig()
	if runtime.GOOS == "darwin" && projectConf["MUTAGEN_USE"] != "false" {
		clearMutagen(projectName, "php")
	}
	composeFileOS := paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/docker-compose.override.yml"
	profilesOn := []string{
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
	cmd := exec.Command("docker-compose", profilesOn...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func UpNginx() {
	cmd := exec.Command("docker-compose", "-f", paths.GetExecDirPath()+"/aruntime/docker-compose.yml", "up", "-d")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		UpNginxWithBuild()
	}
}

func UpNginxWithBuild() {
	dockerComposePull([]string{"-f", paths.GetExecDirPath() + "/aruntime/docker-compose.yml"})
	cmd := exec.Command("docker-compose", "-f", paths.GetExecDirPath()+"/aruntime/docker-compose.yml", "up", "--build", "--force-recreate", "--no-deps", "-d")
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
	composeFileOS := paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/docker-compose.override.yml"
	profilesOn := []string{
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
	dockerComposePull([]string{"-f",
		paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/docker-compose.yml",
		"-f",
		composeFileOS})
	cmd := exec.Command("docker-compose", profilesOn...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	projectConfig := configs.GetCurrentProjectConfig()
	if runtime.GOOS == "darwin" && projectConfig["MUTAGEN_USE"] != "false" {
		syncMutagen(projectName, "php", "www-data")
	}

	if val, ok := projectConfig["CRON_ENABLED"]; ok && val == "true" {
		Cron("on", false)
	} else {
		Cron("off", false)
	}
}

func dockerComposePull(composeFiles []string) {
	composeFiles = append(composeFiles, "pull")
	cmd := exec.Command("docker-compose", composeFiles...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

func syncMutagen(projectName, containerName, usr string) {
	clearMutagen(projectName, containerName)
	cmd := exec.Command("mutagen", "sync", "create", "--name",
		strings.ToLower(projectName)+"-"+containerName+"-1",
		"--default-group-beta", usr,
		"--default-owner-beta", usr,
		"--sync-mode", "two-way-resolved",
		"--default-file-mode", "0664",
		"--default-directory-mode", "0755",
		"--symlink-mode", "posix-raw",
		"--ignore-vcs",
		"-i", "/pub/static",
		"-i", "/pub/media",
		"-i", "/generated",
		"-i", "/var/cache",
		"-i", "/var/view_preprocessed",
		"-i", "/var/page_cache",
		"-i", "/var/tmp",
		"-i", "/var/vendor",
		"-i", "/var/composer_home",
		"-i", "/phpserver",
		"-i", "/.idea",
		paths.GetRunDirPath(),
		"docker://"+strings.ToLower(projectName)+"-"+containerName+"-1/var/www/html",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	} else {
		fmt.Println("Synchronization enabled")
	}
}

func clearMutagen(projectName, containerName string) {
	cmd := exec.Command("mutagen", "sync", "terminate",
		projectName+"-"+containerName+"-1",
	)
	cmd.Run()
}

func DownNginx() {
	composeFile := paths.GetExecDirPath() + "/aruntime/docker-compose.yml"
	if _, err := os.Stat(composeFile); !os.IsNotExist(err) {
		cmd := exec.Command("docker-compose", "-f", composeFile, "down")
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
		cmd := exec.Command("docker-compose", "-f", composeFile, "stop")
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

func Cron(flag string, manual bool) {
	projectName := paths.GetRunDirName()
	var cmd *exec.Cmd
	var bOut io.Writer
	var bErr io.Writer
	if flag == "on" {
		cmdSub := exec.Command("docker", "exec", "-i", "-u", "www-data", strings.ToLower(projectName)+"-php-1", "bash", "-c", "cd /var/www/html && php bin/magento cron:install &&  php bin/magento cron:run")
		cmdSub.Stdout = os.Stdout
		cmdSub.Stderr = os.Stderr
		err := cmdSub.Run()
		if err != nil {
			log.Fatal(err)
		}

		cmd = exec.Command("docker", "exec", "-i", "-u", "root", strings.ToLower(projectName)+"-php-1", "service", "cron", "start")
		cmd.Stdout = bOut
		cmd.Stderr = bErr
		err = cmd.Run()
		if manual {
			if err != nil {
				fmt.Println(bErr)
				log.Fatal(err)
			} else {
				fmt.Println("Cron was started")
			}
		}
	} else {
		cmd = exec.Command("docker", "exec", "-i", "-u", "root", strings.ToLower(projectName)+"-php-1", "service", "cron", "status")
		cmd.Stdout = bOut
		cmd.Stderr = bErr
		err := cmd.Run()
		if err == nil {
			cmdSub := exec.Command("docker", "exec", "-i", "-u", "www-data", strings.ToLower(projectName)+"-php-1", "bash", "-c", "cd /var/www/html && php bin/magento cron:remove")
			cmdSub.Stdout = bOut
			cmdSub.Stderr = bErr
			err := cmdSub.Run()
			if manual {
				if err != nil {
					fmt.Println(bErr)
					log.Fatal(err)
				} else {
					fmt.Println("Cron was removed from Magento")
				}
			}

			cmd = exec.Command("docker", "exec", "-i", "-u", "root", strings.ToLower(projectName)+"-php-1", "service", "cron", "stop")
			cmd.Stdout = bOut
			cmd.Stderr = bErr
			err = cmd.Run()
			if manual {
				if err != nil {
					fmt.Println(bErr)
					log.Fatal(err)
				} else {
					fmt.Println("Cron was stopped from System (container)")
				}
			}
		}
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
	cmd := exec.Command("docker-compose", "-f", paths.GetExecDirPath()+"/aruntime/projects/"+projectName+"/docker-compose.yml", "run", "--rm", "--service-ports", "node", "bash", "-c", "cd /var/www/html && "+flag)
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
