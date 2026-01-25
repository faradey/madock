package docker

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"

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

		// First, install jobs from config (for all platforms)
		installCronJobsFromConfig(projectConf, projectName, manual)

		// Then, platform-specific cron setup
		if projectConf["platform"] == "magento2" {
			cmdSub := exec.Command("docker", "exec", "-i", "-u", "www-data", GetContainerName(projectConf, projectName, "php"), "bash", "-c", "cd "+projectConf["workdir"]+" && php bin/magento cron:install && php bin/magento cron:run")
			cmdSub.Stdout = os.Stdout
			cmdSub.Stderr = os.Stderr
			err = cmdSub.Run()
			if err != nil {
				logger.Println(err)
				fmtc.WarningLn(err.Error())
			}
		} else if projectConf["platform"] == "shopify" {
			containerName := GetContainerName(projectConf, projectName, "php")
			fmt.Println("Setting up Shopify cron...")
			fmt.Printf("  Container: %s\n", containerName)
			fmt.Printf("  Workdir: %s\n", projectConf["workdir"])
			fmt.Println("  Searching for artisan file...")

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
			// First, remove config-based jobs (for all platforms)
			removeCronJobsFromConfig(projectConf, projectName, manual)

			// Then, platform-specific cron removal
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

// getCronJobsFromConfig extracts cron jobs from project configuration
func getCronJobsFromConfig(projectConf map[string]string) []string {
	var jobs []string
	jobsMap := make(map[string]string)

	// Collect all cron/jobs/* entries
	for key, value := range projectConf {
		if strings.HasPrefix(key, "cron/jobs/") && value != "" {
			jobsMap[key] = value
		}
	}

	// Sort keys for consistent order
	keys := make([]string, 0, len(jobsMap))
	for key := range jobsMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		jobs = append(jobs, jobsMap[key])
	}

	return jobs
}

// installCronJobsFromConfig installs cron jobs from configuration
func installCronJobsFromConfig(projectConf map[string]string, projectName string, manual bool) {
	jobs := getCronJobsFromConfig(projectConf)
	if len(jobs) == 0 {
		if manual {
			fmt.Println("No cron jobs defined in configuration")
		}
		return
	}

	containerName := GetContainerName(projectConf, projectName, "php")

	// First, remove existing crontab for www-data user
	cmdRemove := exec.Command("docker", "exec", "-i", "-u", "root", containerName, "crontab", "-u", "www-data", "-r")
	_ = cmdRemove.Run() // Ignore error if no crontab exists

	// Build crontab content
	crontabContent := strings.Join(jobs, "\n") + "\n"

	// Install crontab for www-data user
	cmdSub := exec.Command("docker", "exec", "-i", "-u", "root", containerName, "bash", "-c",
		fmt.Sprintf("echo '%s' | crontab -u www-data -", crontabContent))
	cmdSub.Stdout = os.Stdout
	cmdSub.Stderr = os.Stderr
	err := cmdSub.Run()

	if manual {
		if err != nil {
			logger.Println(err)
			fmtc.WarningLn(err.Error())
		} else {
			fmtc.SuccessLn(fmt.Sprintf("Installed %d cron job(s)", len(jobs)))
		}
	}
}

// removeCronJobsFromConfig removes cron jobs installed from configuration
func removeCronJobsFromConfig(projectConf map[string]string, projectName string, manual bool) {
	containerName := GetContainerName(projectConf, projectName, "php")

	// Remove crontab for www-data user
	cmdSub := exec.Command("docker", "exec", "-i", "-u", "root", containerName, "crontab", "-u", "www-data", "-r")
	cmdSub.Stdout = os.Stdout
	cmdSub.Stderr = os.Stderr
	err := cmdSub.Run()

	if manual {
		if err != nil {
			// crontab -r returns error if no crontab exists, which is fine
			fmt.Println("Cron jobs removed (or none existed)")
		} else {
			fmtc.SuccessLn("Cron jobs removed")
		}
	}
}
