package builder

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/faradey/madock/src/configs"
)

func Cron(flag, manual bool) {
	projectName := configs.GetProjectName()
	projectConfig := configs.GetCurrentProjectConfig()
	var cmd *exec.Cmd
	var bOut io.Writer
	var bErr io.Writer
	if flag {
		cmd = exec.Command("docker", "exec", "-i", "-u", "root", strings.ToLower(projectConfig["CONTAINER_NAME_PREFIX"])+strings.ToLower(projectName)+"-php-1", "service", "cron", "start")
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

		cmdSub := exec.Command("docker", "exec", "-i", "-u", "www-data", strings.ToLower(projectConfig["CONTAINER_NAME_PREFIX"])+strings.ToLower(projectName)+"-php-1", "bash", "-c", "cd "+projectConfig["WORKDIR"]+" && php bin/magento cron:remove && php bin/magento cron:install && php bin/magento cron:run")
		cmdSub.Stdout = os.Stdout
		cmdSub.Stderr = os.Stderr
		err = cmdSub.Run()
		if err != nil {
			log.Fatal(err)
		}
	} else {
		cmd = exec.Command("docker", "exec", "-i", "-u", "root", strings.ToLower(projectConfig["CONTAINER_NAME_PREFIX"])+strings.ToLower(projectName)+"-php-1", "service", "cron", "status")
		cmd.Stdout = bOut
		cmd.Stderr = bErr
		err := cmd.Run()
		if err == nil {
			cmdSub := exec.Command("docker", "exec", "-i", "-u", "www-data", strings.ToLower(projectConfig["CONTAINER_NAME_PREFIX"])+strings.ToLower(projectName)+"-php-1", "bash", "-c", "cd "+projectConfig["WORKDIR"]+" && php bin/magento cron:remove")
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

			cmd = exec.Command("docker", "exec", "-i", "-u", "root", strings.ToLower(projectConfig["CONTAINER_NAME_PREFIX"])+strings.ToLower(projectName)+"-php-1", "service", "cron", "stop")
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
