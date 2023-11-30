package docker

import (
	"fmt"
	cliHelper "github.com/faradey/madock/src/helper/cli"
	configs2 "github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/configs/aruntime/nginx"
	"github.com/faradey/madock/src/helper/configs/aruntime/project"
	"github.com/faradey/madock/src/helper/hash"
	"github.com/faradey/madock/src/helper/paths"
	"github.com/gosimple/hashdir"
	"io"
	"log"
	"os"
	"os/exec"
	"os/user"
	"strings"
)

func UpWithBuild(withChown bool) {
	UpNginx()
	UpProjectWithBuild(withChown)
}

func Down(withVolumes bool) {
	projectName := configs2.GetProjectName()
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

func Kill() {
	projectName := configs2.GetProjectName()
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

func UpNginx() {
	UpNginxWithBuild()
}

func UpNginxWithBuild() {
	projectName := configs2.GetProjectName()
	nginx.MakeConf()
	project.MakeConf(projectName)
	projectConf := configs2.GetCurrentProjectConfig()
	dirHash, err := hashdir.Make(paths.GetExecDirPath()+"/aruntime/projects/"+projectName+"/ctx", "md5")
	dockerComposeHash, err := hash.HashFile(paths.GetExecDirPath()+"/aruntime/projects/"+projectName+"/docker-compose.yml", "md5")
	dockerComposeOverHash, err := hash.HashFile(paths.GetExecDirPath()+"/aruntime/projects/"+projectName+"/docker-compose.override.yml", "md5")
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
		configs2.SetParam(envFile, "CACHE_HASH", dirHash)
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

func UpProjectWithBuild(withChown bool) {
	projectName := configs2.GetProjectName()
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

	projectConf := configs2.GetCurrentProjectConfig()

	if val, ok := projectConf["CRON_ENABLED"]; ok && val == "true" {
		CronExecute(true, false)
	} else {
		CronExecute(false, false)
	}

	if withChown {
		usr, _ := user.Current()
		cmd := exec.Command("docker", "exec", "-it", "-u", "root", GetContainerName(projectConf, projectName, "php"), "bash", "-c", "chown -R "+usr.Uid+":"+usr.Gid+" "+projectConf["WORKDIR"]+" && chown -R "+usr.Uid+":"+usr.Gid+" /var/www/.composer")
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

func DownNginx(force bool) {
	composeFile := paths.GetExecDirPath() + "/aruntime/docker-compose.yml"
	if _, err := os.Stat(composeFile); !os.IsNotExist(err) {
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
	if _, err := os.Stat(composeFile); !os.IsNotExist(err) {
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

func GetContainerName(projectConf map[string]string, projectName, service string) string {
	return strings.ToLower(projectConf["CONTAINER_NAME_PREFIX"]) + strings.ToLower(projectName) + "-" + service + "-1"
}

func CronExecute(flag, manual bool) {
	projectName := configs2.GetProjectName()
	projectConf := configs2.GetCurrentProjectConfig()
	service := "php"
	if projectConf["PLATFORM"] == "pwa" {
		service = "nodejs"
	}

	service, user, _ := cliHelper.GetEnvForUserServiceWorkdir(service, "root", "")

	var cmd *exec.Cmd
	var bOut io.Writer
	var bErr io.Writer
	if flag {
		cmd = exec.Command("docker", "exec", "-i", "-u", user, GetContainerName(projectConf, projectName, service), "service", "cron", "start")
		cmd.Stdout = bOut
		cmd.Stderr = bErr
		err := cmd.Run()
		if manual {
			if err != nil {
				fmt.Println(bErr)
				log.Fatal(err)
			} else {
				fmt.Println("Cron was started")
			}
		}

		if projectConf["PLATFORM"] == "magento2" {
			cmdSub := exec.Command("docker", "exec", "-i", "-u", "www-data", GetContainerName(projectConf, projectName, "php"), "bash", "-c", "cd "+projectConf["WORKDIR"]+" && php bin/magento cron:remove && php bin/magento cron:install && php bin/magento cron:run")
			cmdSub.Stdout = os.Stdout
			cmdSub.Stderr = os.Stderr
			err = cmdSub.Run()
			if err != nil {
				log.Fatal(err)
			}
		}
	} else {
		cmd = exec.Command("docker", "exec", "-i", "-u", user, GetContainerName(projectConf, projectName, "php"), "service", "cron", "status")
		cmd.Stdout = bOut
		cmd.Stderr = bErr
		err := cmd.Run()
		if err == nil {
			if projectConf["PLATFORM"] == "magento2" {
				cmdSub := exec.Command("docker", "exec", "-i", "-u", "www-data", GetContainerName(projectConf, projectName, "php"), "bash", "-c", "cd "+projectConf["WORKDIR"]+" && php bin/magento cron:remove")
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
			}

			cmd = exec.Command("docker", "exec", "-i", "-u", user, GetContainerName(projectConf, projectName, "php"), "service", "cron", "stop")
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
