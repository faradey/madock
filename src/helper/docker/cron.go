package docker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	cliHelper "github.com/faradey/madock/v3/src/helper/cli"
	"github.com/faradey/madock/v3/src/helper/cli/fmtc"
	configs2 "github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/logger"
)

// containerExecSilent runs a command in the container while capturing stdout/stderr
// instead of streaming them to the user's terminal. Returns combined output and error.
func containerExecSilent(container, user string, command ...string) (string, error) {
	cmd, err := PrepareContainerExec(container, user, false, command...)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	execErr := cmd.Run()
	NotifyExecDone(container, command, execErr)
	return buf.String(), execErr
}

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
			containerName := GetContainerName(projectConf, projectName, "php")
			workdir := projectConf["workdir"]
			// `cron:install` is NOT idempotent: if the `#~ MAGENTO START ... #~ MAGENTO END`
			// block is already present in www-data's crontab, it prints
			// `Crontab has already been generated and saved` and exits with status 1.
			// `cron:install --force` would handle this, but the `--force` flag is not
			// available in older Magento versions that madock still supports (2.0.x–2.1.x).
			// Run `cron:remove` first instead — it is idempotent across all versions and
			// has been the proven sequence in madock from 2022 until commit ca6a668.
			cmd := "cd " + workdir + " && php bin/magento cron:remove && php bin/magento cron:install && php bin/magento cron:run"
			if manual {
				err = ContainerExec(containerName, "www-data", false, "bash", "-c", cmd)
				if err != nil {
					logger.Println(err)
					fmtc.WarningLn(err.Error())
				}
			} else {
				// Auto-invocation (start/rebuild). On a fresh container the Magento DI
				// can be in a "warming up" state for the first several seconds:
				// `generated/code` + `generated/metadata` are being lazily (re)compiled,
				// and during that window `bin/magento` may not yet expose `cron:*`
				// commands and prints `There are no commands defined in the "cron"
				// namespace`. Probing once and warning would be a false positive — wait
				// for the namespace to settle, then act.
				probe := "cd " + workdir + " && php bin/magento list cron --format=txt 2>&1"
				const probeAttempts = 6
				const probeDelay = 2 * time.Second
				var probeOut string
				namespaceReady := false
				for attempt := 1; attempt <= probeAttempts; attempt++ {
					probeOut, _ = containerExecSilent(containerName, "www-data", "bash", "-c", probe)
					if !strings.Contains(probeOut, "There are no commands defined") {
						namespaceReady = true
						break
					}
					if attempt < probeAttempts {
						time.Sleep(probeDelay)
					}
				}

				if !namespaceReady {
					fmtc.WarningLn("Magento cron commands are not registered in this instance — scheduled jobs will NOT run.")
					fmtc.WarningLn("Likely cause: stale compiled DI in generated/. To fix, run inside the project (php container):")
					fmtc.WarningLn("  rm -rf generated/code generated/metadata var/cache/* var/page_cache/*")
					fmtc.WarningLn("  bin/magento setup:upgrade")
					fmtc.WarningLn("  bin/magento setup:di:compile   # only if you use production mode")
					logger.Println(fmt.Sprintf("magento2 cron: cron:* namespace still empty after %d attempts; cron:install/cron:run skipped. Last probe output:\n%s", probeAttempts, probeOut))
				} else {
					out, cerr := containerExecSilent(containerName, "www-data", "bash", "-c", cmd)
					if cerr != nil {
						fmtc.WarningLn("Magento cron setup failed — scheduled jobs may NOT run. See debug.log for details.")
						logger.Println(cerr)
						if out != "" {
							logger.Println(out)
						}
					}
				}
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
		_, err := containerExecSilent(GetContainerName(projectConf, projectName, service), userOS, "service", "cron", "status")
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
