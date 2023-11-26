package builder

import (
	"fmt"
	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/configs/aruntime/nginx"
	"github.com/faradey/madock/src/configs/aruntime/project"
	"github.com/faradey/madock/src/controller/general/cron"
	"github.com/faradey/madock/src/helper"
	"github.com/faradey/madock/src/helper/paths"
	"github.com/gosimple/hashdir"
	"log"
	"os"
	"os/exec"
	"os/user"
	"strings"
)

func UpWithBuild(withChown bool) {
	DownNginx()
	UpNginx()
	upProjectWithBuild(withChown)
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

	projectConf := configs.GetCurrentProjectConfig()

	if val, ok := projectConf["CRON_ENABLED"]; ok && val == "true" {
		cron.Execute(true, false)
	} else {
		cron.Execute(false, false)
	}

	if withChown {
		usr, _ := user.Current()
		cmd := exec.Command("docker", "exec", "-it", "-u", "root", strings.ToLower(projectConf["CONTAINER_NAME_PREFIX"])+strings.ToLower(projectName)+"-php-1", "bash", "-c", "chown -R "+usr.Uid+":"+usr.Gid+" "+projectConf["WORKDIR"]+" && chown -R "+usr.Uid+":"+usr.Gid+" /var/www/.composer")
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

func DownloadMagento(projectName, edition, version string, isSampleData bool) {
	projectConf := configs.GetCurrentProjectConfig()
	sampleData := ""
	if isSampleData {
		sampleData = " && bin/magento sampledata:deploy"
	}
	command := []string{
		"exec",
		"-it",
		"-u",
		"www-data",
		strings.ToLower(projectConf["CONTAINER_NAME_PREFIX"]) + strings.ToLower(projectName) + "-php-1",
		"bash",
		"-c",
		"cd " + projectConf["WORKDIR"] + " " +
			"&& rm -r -f " + projectConf["WORKDIR"] + "/download-magento123456789 " +
			"&& mkdir " + projectConf["WORKDIR"] + "/download-magento123456789 " +
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
