package docker

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"

	cliHelper "github.com/faradey/madock/src/helper/cli"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	configs2 "github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/logger"
)

// CronExecute starts or stops cron service in the container
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
			containerName := GetContainerName(projectConf, projectName, "php")
			publicDir := projectConf["public_dir"]
			if publicDir == "" {
				publicDir = "web"
			}
			fmt.Println("Setting up Shopify cron...")
			fmt.Printf("  Container: %s\n", containerName)
			fmt.Printf("  Workdir: %s\n", projectConf["workdir"])
			fmt.Printf("  Public dir: %s\n", publicDir)
			fmt.Printf("  Script: /var/www/scripts/php/shopify-crontab.php\n")
			fmt.Printf("  Cron job: * * * * * cd %s/%s && php artisan schedule:run\n", projectConf["workdir"], publicDir)

			data, err := json.Marshal(projectConf)
			if err != nil {
				logger.Fatal(err)
			}

			conf := string(data)
			cmdSub := exec.Command("docker", "exec", "-i", "-u", "www-data", containerName, "php", "/var/www/scripts/php/shopify-crontab.php", conf, "0")
			cmdSub.Stdout = os.Stdout
			cmdSub.Stderr = os.Stderr
			err = cmdSub.Run()
			if err != nil {
				logger.Println(err)
				fmtc.WarningLn(err.Error())
			} else {
				fmtc.SuccessLn("Shopify cron job installed successfully")
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
				containerName := GetContainerName(projectConf, projectName, "php")
				fmt.Println("Removing Shopify cron...")
				fmt.Printf("  Container: %s\n", containerName)
				fmt.Printf("  Script: /var/www/scripts/php/shopify-crontab.php\n")

				data, err := json.Marshal(projectConf)
				if err != nil {
					logger.Fatal(err)
				}

				conf := string(data)
				cmdSub := exec.Command("docker", "exec", "-i", "-u", "www-data", containerName, "php", "/var/www/scripts/php/shopify-crontab.php", conf, "1")
				cmdSub.Stdout = os.Stdout
				cmdSub.Stderr = os.Stderr
				err = cmdSub.Run()
				if manual {
					if err != nil {
						logger.Println(err)
						fmtc.WarningLn(err.Error())
					} else {
						fmtc.SuccessLn("Shopify cron job removed successfully")
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
