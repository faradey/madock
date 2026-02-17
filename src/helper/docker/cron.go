package docker

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	cliHelper "github.com/faradey/madock/v3/src/helper/cli"
	"github.com/faradey/madock/v3/src/helper/cli/fmtc"
	configs2 "github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/logger"
)

// CronExecute starts or stops cron service in the container
func CronExecute(projectName string, flag, manual bool) {
	projectConf := configs2.GetProjectConfig(projectName)
	service := resolveMainService(projectConf)

	service, userOS, _ := cliHelper.GetEnvForUserServiceWorkdir(service, "root", "")

	if flag {
		err := ContainerExec(GetContainerName(projectConf, projectName, service), userOS, false, "service", "cron", "start")
		if manual {
			if err != nil {
				logger.Fatal(err)
			} else {
				fmt.Println("Cron was started")
			}
		}

		// First, install jobs from config (for all platforms)
		installCronJobsFromConfig(projectConf, projectName, manual)

		// Then, platform-specific cron setup
		if projectConf["platform"] == "magento2" {
			err = ContainerExec(GetContainerName(projectConf, projectName, "php"), "www-data", false, "bash", "-c", "cd "+projectConf["workdir"]+" && php bin/magento cron:install && php bin/magento cron:run")
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
			err = ContainerExec(containerName, "www-data", false, "php", "/var/www/scripts/php/shopify-crontab.php", conf, "0")
			if err != nil {
				logger.Println(err)
				fmtc.WarningLn(err.Error())
			} else {
				fmtc.SuccessLn("Shopify cron job installed successfully")
			}
		}
	} else {
		err := ContainerExec(GetContainerName(projectConf, projectName, service), userOS, false, "service", "cron", "status")
		if err == nil {
			// First, remove config-based jobs (for all platforms)
			removeCronJobsFromConfig(projectConf, projectName, manual)

			// Then, platform-specific cron removal
			if projectConf["platform"] == "magento2" {
				err := ContainerExec(GetContainerName(projectConf, projectName, "php"), "www-data", false, "bash", "-c", "cd "+projectConf["workdir"]+" && php bin/magento cron:remove")
				if manual {
					if err != nil {
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
				err = ContainerExec(containerName, "www-data", false, "php", "/var/www/scripts/php/shopify-crontab.php", conf, "1")
				if manual {
					if err != nil {
						logger.Println(err)
						fmtc.WarningLn(err.Error())
					} else {
						fmtc.SuccessLn("Shopify cron job removed successfully")
					}
				}
			}

			err = ContainerExec(GetContainerName(projectConf, projectName, service), userOS, false, "service", "cron", "stop")
			if manual {
				if err != nil {
					logger.Fatal(err)
				} else {
					fmt.Println("Cron was stopped from System (container)")
				}
			}
		}
	}
}

// resolveMainService determines the main service name based on project config.
// This is a local helper to avoid importing the platform package (which would cause an import cycle).
func resolveMainService(projectConf map[string]string) string {
	if lang, ok := projectConf["language"]; ok && lang != "" && lang != "php" {
		switch lang {
		case "nodejs":
			return "nodejs"
		case "python":
			return "python"
		case "golang":
			return "golang"
		case "ruby":
			return "ruby"
		case "none":
			return "app"
		}
	}
	return "php"
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

	containerName := GetContainerName(projectConf, projectName, resolveMainService(projectConf))

	// First, remove existing crontab
	removeCronJobsFromConfig(projectConf, projectName, false)

	// Build crontab content
	crontabContent := strings.Join(jobs, "\n") + "\n"

	// Install crontab for www-data user
	err := ContainerExec(containerName, "root", false, "bash", "-c",
		fmt.Sprintf("echo '%s' | crontab -u www-data -", crontabContent))

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
	containerName := GetContainerName(projectConf, projectName, resolveMainService(projectConf))

	// Remove crontab for www-data user
	err := ContainerExec(containerName, "root", false, "crontab", "-u", "www-data", "-r")

	if manual {
		if err != nil {
			// crontab -r returns error if no crontab exists, which is fine
			fmt.Println("Cron jobs removed (or none existed)")
		} else {
			fmtc.SuccessLn("Cron jobs removed")
		}
	}
}
