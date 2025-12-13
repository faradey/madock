package docker

import (
	"encoding/json"
	"fmt"
	cliHelper "github.com/faradey/madock/src/helper/cli"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	configs2 "github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/configs/aruntime/nginx"
	"github.com/faradey/madock/src/helper/configs/aruntime/project"
	"github.com/faradey/madock/src/helper/logger"
	"github.com/faradey/madock/src/helper/paths"
	"io"
	"os"
	"os/exec"
	"os/user"
	"strings"
)

func UpWithBuild(projectName string, withChown bool) {
	UpNginxWithBuild(projectName, true)
	UpProjectWithBuild(projectName, withChown)
}

func Down(projectName string, withVolumes bool) {
	composeFile := paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/docker-compose.yml"
	composeFileOS := paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/docker-compose.override.yml"
	if paths.IsFileExist(composeFile) {
		profilesOn := []string{
			"compose",
			"-f",
			composeFile,
			"-f",
			composeFileOS,
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

func Kill(projectName string) {
	composeFile := paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/docker-compose.yml"
	composeFileOS := paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/docker-compose.override.yml"
	if paths.IsFileExist(composeFile) {
		profilesOn := []string{
			"compose",
			"-f",
			composeFile,
			"-f",
			composeFileOS,
		}

		profilesOn = append(profilesOn, "kill")

		cmd := exec.Command("docker", profilesOn...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			fmt.Println(err)
		}
	}
}

func UpNginx(projectName string) {
	UpNginxWithBuild(projectName, false)
}

func UpNginxWithBuild(projectName string, force bool) {
	if !paths.IsFileExist(paths.GetRunDirPath() + "/.madock/config.xml") {
		configs2.SetParam(configs2.MadockLevelConfigCode, "path", paths.GetRunDirPath(), "default", configs2.MadockLevelConfigCode)
	}
	nginx.MakeConf(projectName)
	project.MakeConf(projectName)
	projectConf := configs2.GetProjectConfig(projectName)
	doNeedRunAruntime := true
	if paths.IsFileExist(paths.GetExecDirPath() + "/aruntime/docker-compose.yml") {
		cmd := exec.Command("docker", "compose", "-f", paths.GetExecDirPath()+"/aruntime/docker-compose.yml", "ps", "--format", "json")
		result, err := cmd.CombinedOutput()
		if err != nil {
			logger.Println(err, result)
		} else {
			if len(result) > 100 && strings.Contains(string(result), "\"Command\"") && strings.Contains(string(result), "\"aruntime-nginx\"") {
				doNeedRunAruntime = false
			}
		}
	}

	if (!paths.IsFileExist(paths.GetExecDirPath()+"/cache/conf-cache") || doNeedRunAruntime) && projectConf["proxy/enabled"] == "true" {
		ctxPath := paths.MakeDirsByPath(paths.GetExecDirPath() + "/aruntime/ctx")
		if !paths.IsFileExist(paths.GetExecDirPath() + "/cache/conf-cache") {
			nginx.GenerateSslCert(ctxPath, false)

			dockerComposePull([]string{"compose", "-f", paths.GetExecDirPath() + "/aruntime/docker-compose.yml"})

			err := os.WriteFile(paths.GetExecDirPath()+"/cache/conf-cache", []byte("config cache"), 0755)
			if err != nil {
				logger.Fatal(err)
			}
		}
		command := []string{"compose", "-f", paths.GetExecDirPath() + "/aruntime/docker-compose.yml", "up", "--no-deps", "-d"}
		if force {
			command = append(command, "--build", "--force-recreate")
		}
		cmd := exec.Command("docker", command...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			logger.Println(err)
		}
	}
}

func UpProjectWithBuild(projectName string, withChown bool) {
	var err error
	if !paths.IsFileExist(paths.GetExecDirPath() + "/aruntime/.composer") {
		err = os.Chmod(paths.MakeDirsByPath(paths.GetExecDirPath()+"/aruntime/.composer"), 0777)
		if err != nil {
			logger.Fatal(err)
		}
	}

	composerGlobalDir, err := os.UserHomeDir()
	if err != nil {
		logger.Fatal(err)
	} else {
		if !paths.IsFileExist(composerGlobalDir + "/.composer") {
			paths.MakeDirsByPath(composerGlobalDir + "/.composer")
		}
	}

	src := paths.MakeDirsByPath(paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/composer")

	if fi, err := os.Lstat(src); err == nil {
		if fi.Mode()&os.ModeSymlink != os.ModeSymlink {
			err = os.RemoveAll(src)
			if err == nil {
				err = os.Symlink(composerGlobalDir+"/.composer", src)
				if err != nil {
					logger.Fatal(err)
				}
			} else {
				fmt.Println(err)
			}
		}
	} else {
		err = os.Symlink(composerGlobalDir+"/.composer", src)
		if err != nil {
			logger.Fatal(err)
		}
	}

	sshDir := paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/ssh"

	if fi, err := os.Lstat(sshDir); err == nil {
		if fi.Mode()&os.ModeSymlink != os.ModeSymlink {
			err = os.RemoveAll(sshDir)
			if err == nil {
				err = os.Symlink(composerGlobalDir+"/.ssh", sshDir)
				if err != nil {
					logger.Fatal(err)
				}
			} else {
				fmt.Println(err)
			}
		}
	} else {
		err = os.Symlink(composerGlobalDir+"/.ssh", sshDir)
		if err != nil {
			logger.Fatal(err)
		}
	}

	composeFile := paths.MakeDirsByPath(paths.GetExecDirPath()+"/aruntime/projects/"+projectName) + "/docker-compose.yml"
	composeFileOS := paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/docker-compose.override.yml"
	profilesOn := []string{
		"compose",
		"-f",
		composeFile,
		"-f",
		composeFileOS,
		"up",
		"--build",
		"--force-recreate",
		"--no-deps",
		"-d",
	}
	dockerComposePull([]string{"compose", "-f", composeFile, "-f", composeFileOS})
	cmd := exec.Command("docker", profilesOn...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		logger.Fatal(err)
	}

	projectConf := configs2.GetProjectConfig(projectName)

	if val, ok := projectConf["cron/enabled"]; ok && val == "true" {
		CronExecute(projectName, true, false)
	} else {
		CronExecute(projectName, false, false)
	}

	if withChown {
		usr, _ := user.Current()
		cmd := exec.Command("docker", "exec", "-it", "-u", "root", GetContainerName(projectConf, projectName, "php"), "bash", "-c", "chown -R "+usr.Uid+":"+usr.Gid+" "+projectConf["workdir"]+" && chown -R "+usr.Uid+":"+usr.Gid+" /var/www/.composer")
		/* for .npm for futures +" && chown -R "+usr.Uid+":"+usr.Gid+" /var/www/.npm" */
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			logger.Fatal(err)
		}
	}
}

func dockerComposePull(composeFiles []string) {
	composeFiles = append(composeFiles, "pull")
	cmd := exec.Command("docker", composeFiles...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		logger.Fatal(err)
	}
}

func DownNginx(force bool) {
	composeFile := paths.GetExecDirPath() + "/aruntime/docker-compose.yml"
	if paths.IsFileExist(composeFile) {
		command := "down"
		if force {
			command = "kill"
		}
		cmd := exec.Command("docker", "compose", "-f", composeFile, command)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			fmt.Println(err)
		}
	}
}

func StopNginx(force bool) {
	composeFile := paths.GetExecDirPath() + "/aruntime/docker-compose.yml"
	if paths.IsFileExist(composeFile) {
		command := "stop"
		if force {
			command = "kill"
		}
		cmd := exec.Command("docker", "compose", "-f", composeFile, command)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			fmt.Println(err)
		}
	}
}

func ReloadNginx() {
	cmd := exec.Command("docker", "exec", "aruntime-nginx", "nginx", "-s", "reload")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmt.Println(err)
	}
}

func GetContainerName(projectConf map[string]string, projectName, service string) string {
	scope := ""
	if val, ok := projectConf["activeScope"]; ok && val != "default" {
		scope = strings.ToLower("-" + val)
	}
	return strings.ToLower(projectConf["container_name_prefix"]) + strings.ToLower(projectName) + scope + "-" + service + "-1"
}

func CronExecute(projectName string, flag, manual bool) {
	projectConf := configs2.GetProjectConfig(projectName)
	service := "php"
	if projectConf["platform"] == "pwa" {
		service = "nodejs"
	}

	service, userOS, _ := cliHelper.GetEnvForUserServiceWorkdir(service, "root", "")

	var cmd *exec.Cmd
	var bOut io.Writer
	var bErr io.Writer
	if flag {
		cmd = exec.Command("docker", "exec", "-i", "-u", userOS, GetContainerName(projectConf, projectName, service), "service", "cron", "start")
		cmd.Stdout = bOut
		cmd.Stderr = bErr
		err := cmd.Run()
		if manual {
			if err != nil {
				fmt.Println(bErr)
				logger.Fatal(err)
			} else {
				fmt.Println("Cron was started")
			}
		}

		if projectConf["platform"] == "magento2" {
			cmdSub := exec.Command("docker", "exec", "-i", "-u", "www-data", GetContainerName(projectConf, projectName, "php"), "bash", "-c", "cd "+projectConf["workdir"]+" && php bin/magento cron:remove && php bin/magento cron:install && php bin/magento cron:run")
			cmdSub.Stdout = os.Stdout
			cmdSub.Stderr = os.Stderr
			err = cmdSub.Run()
			if err != nil {
				logger.Println(err)
				fmtc.WarningLn(err.Error())
			}
		} else if projectConf["platform"] == "shopify" {
			data, err := json.Marshal(projectConf)
			if err != nil {
				logger.Fatal(err)
			}

			conf := string(data)
			cmdSub := exec.Command("docker", "exec", "-i", "-u", "www-data", GetContainerName(projectConf, projectName, "php"), "php", "/var/www/scripts/php/shopify-crontab.php", conf, "0")
			cmdSub.Stdout = os.Stdout
			cmdSub.Stderr = os.Stderr
			err = cmdSub.Run()
			if err != nil {
				logger.Println(err)
				fmtc.WarningLn(err.Error())
			}

		}
	} else {
		cmd = exec.Command("docker", "exec", "-i", "-u", userOS, GetContainerName(projectConf, projectName, "php"), "service", "cron", "status")
		cmd.Stdout = bOut
		cmd.Stderr = bErr
		err := cmd.Run()
		if err == nil {
			if projectConf["platform"] == "magento2" {
				cmdSub := exec.Command("docker", "exec", "-i", "-u", "www-data", GetContainerName(projectConf, projectName, "php"), "bash", "-c", "cd "+projectConf["workdir"]+" && php bin/magento cron:remove")
				cmdSub.Stdout = bOut
				cmdSub.Stderr = bErr
				err := cmdSub.Run()
				if manual {
					if err != nil {
						logger.Println(bErr)
						logger.Println(err)
					} else {
						fmt.Println("Cron was removed from Magento")
					}
				}
			} else if projectConf["platform"] == "shopify" {
				data, err := json.Marshal(projectConf)
				if err != nil {
					logger.Fatal(err)
				}

				conf := string(data)
				cmdSub := exec.Command("docker", "exec", "-i", "-u", "www-data", GetContainerName(projectConf, projectName, "php"), "php", "/var/www/scripts/php/shopify-crontab.php", conf, "1")
				cmdSub.Stdout = bOut
				cmdSub.Stderr = bErr
				err = cmdSub.Run()
				if manual {
					if err != nil {
						logger.Println(bErr)
						logger.Println(err)
					} else {
						fmt.Println("Cron was removed from Shopify")
					}
				}
			}

			cmd = exec.Command("docker", "exec", "-i", "-u", userOS, GetContainerName(projectConf, projectName, "php"), "service", "cron", "stop")
			cmd.Stdout = bOut
			cmd.Stderr = bErr
			err = cmd.Run()
			if manual {
				if err != nil {
					fmt.Println(bErr)
					logger.Fatal(err)
				} else {
					fmt.Println("Cron was stopped from System (container)")
				}
			}
		}
	}
}

func UpSnapshot(projectName string) {
	composerFile := paths.MakeDirsByPath(paths.GetExecDirPath()+"/aruntime/projects/"+projectName) + "/docker-compose-snapshot.yml"
	profilesOn := []string{
		"compose",
		"-f",
		composerFile,
		"up",
		"--build",
		"--force-recreate",
		"--no-deps",
		"-d",
	}
	dockerComposePull([]string{"compose", "-f", composerFile})
	cmd := exec.Command("docker", profilesOn...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		logger.Fatal(err)
	}
}

func StopSnapshot(projectName string) {
	composerFile := paths.MakeDirsByPath(paths.GetExecDirPath()+"/aruntime/projects/"+projectName) + "/docker-compose-snapshot.yml"
	if paths.IsFileExist(composerFile) {
		command := "stop"
		cmd := exec.Command("docker", "compose", "-f", composerFile, command)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			fmt.Println(err)
		}
	}
}
