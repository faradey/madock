package cron

import (
	"fmt"
	"github.com/faradey/madock/src/configs"
	cliHelper "github.com/faradey/madock/src/helper"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
)

func RunCron(flag, manual bool) {
	projectName := configs.GetProjectName()
	projectConf := configs.GetCurrentProjectConfig()
	service := "php"
	if projectConf["PLATFORM"] == "pwa" {
		service = "nodejs"
	}

	service, user, _ := cliHelper.GetUserServiceWorkdir(service, "root", "")

	var cmd *exec.Cmd
	var bOut io.Writer
	var bErr io.Writer
	if flag {
		cmd = exec.Command("docker", "exec", "-i", "-u", user, strings.ToLower(projectConf["CONTAINER_NAME_PREFIX"])+strings.ToLower(projectName)+"-"+service+"-1", "service", "cron", "start")
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

		cmdSub := exec.Command("docker", "exec", "-i", "-u", "www-data", strings.ToLower(projectConf["CONTAINER_NAME_PREFIX"])+strings.ToLower(projectName)+"-php-1", "bash", "-c", "cd "+projectConf["WORKDIR"]+" && php bin/magento cron:remove && php bin/magento cron:install && php bin/magento cron:run")
		cmdSub.Stdout = os.Stdout
		cmdSub.Stderr = os.Stderr
		err = cmdSub.Run()
		if err != nil {
			log.Fatal(err)
		}
	} else {
		cmd = exec.Command("docker", "exec", "-i", "-u", "root", strings.ToLower(projectConf["CONTAINER_NAME_PREFIX"])+strings.ToLower(projectName)+"-php-1", "service", "cron", "status")
		cmd.Stdout = bOut
		cmd.Stderr = bErr
		err := cmd.Run()
		if err == nil {
			cmdSub := exec.Command("docker", "exec", "-i", "-u", "www-data", strings.ToLower(projectConf["CONTAINER_NAME_PREFIX"])+strings.ToLower(projectName)+"-php-1", "bash", "-c", "cd "+projectConf["WORKDIR"]+" && php bin/magento cron:remove")
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

			cmd = exec.Command("docker", "exec", "-i", "-u", "root", strings.ToLower(projectConf["CONTAINER_NAME_PREFIX"])+strings.ToLower(projectName)+"-php-1", "service", "cron", "stop")
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

func Enable() {
	RunCron(true, true)
}

func Disable() {
	RunCron(false, true)
}
