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
	var cmd *exec.Cmd
	var bOut io.Writer
	var bErr io.Writer
	if flag {
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
