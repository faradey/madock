package docker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
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
// CronRunning asks the container whether cron is actually running.
//
// `status` used to answer this from the configuration — cron/enabled — which
// says what was asked for and not what happened. Starting cron is a command
// executed inside the container and it can fail; the setting would still read
// true, and status would report a scheduler that is not there. Anything that
// runs on a schedule then silently does not.
//
// A project that is down answers false, which is correct rather than an error:
// there is no cron running in a container that does not exist.
func CronRunning(projectName string) bool {
	projectConf := configs2.GetProjectConfig(projectName)
	service := resolveMainService(projectConf)
	service, userOS, _ := cliHelper.GetEnvForUserServiceWorkdir(service, "root", "")

	cmd, err := PrepareContainerExec(GetContainerName(projectConf, projectName, service), userOS, false, "service", "cron", "status")
	if err != nil {
		return false
	}
	return cmd.Run() == nil
}

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
		} else if projectConf["platform"] == "shopware" {
			containerName := GetContainerName(projectConf, projectName, "php")
			workdir := projectConf["workdir"]
			// Shopware scheduled-task:run dispatches due scheduled tasks to the
			// messenger queue (which a consumer then executes). We append it to
			// www-data's existing crontab idempotently so it coexists with any
			// jobs already installed via `cron/jobs/*` configuration.
			scheduledLine := fmt.Sprintf("* * * * * cd %s && php bin/console scheduled-task:run --time-limit=60 >/dev/null 2>&1", workdir)
			cmd := fmt.Sprintf(
				"( crontab -u www-data -l 2>/dev/null | grep -v 'scheduled-task:run' || true ; echo '%s' ) | crontab -u www-data -",
				scheduledLine,
			)
			err := ContainerExec(containerName, "root", false, "bash", "-c", cmd)
			if manual {
				if err != nil {
					logger.Println(err)
					fmtc.WarningLn(err.Error())
				} else {
					fmtc.SuccessLn("Shopware scheduled-task cron installed")
				}
			} else if err != nil {
				logger.Println(err)
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
			} else if projectConf["platform"] == "shopware" {
				containerName := GetContainerName(projectConf, projectName, "php")
				// Strip scheduled-task:run from crontab while keeping any other
				// jobs the user may have installed via cron/jobs/*.
				cmd := "( crontab -u www-data -l 2>/dev/null | grep -v 'scheduled-task:run' || true ) | crontab -u www-data -"
				err := ContainerExec(containerName, "root", false, "bash", "-c", cmd)
				if manual {
					if err != nil {
						logger.Println(err)
					} else {
						fmt.Println("Shopware scheduled-task cron was removed")
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

// getCronJobsFromConfig extracts cron jobs from project configuration.
//
// Two spellings reach this map and both are documented: named entries, which
// parse to cron/jobs/<name>, and a repeated <job> tag, which parses to
// cron/jobs/job/<n>.
func getCronJobsFromConfig(projectConf map[string]string) []string {
	var jobs []string
	jobsMap := make(map[string]string)

	// Collect all cron/jobs/* entries
	for key, value := range projectConf {
		if strings.HasPrefix(key, "cron/jobs/") && value != "" {
			jobsMap[key] = value
		}
	}

	// Sort keys for consistent order. Plain string order would read the tenth
	// repeated <job> as the second one, so a numeric segment is compared as a
	// number: job/2 before job/10.
	keys := make([]string, 0, len(jobsMap))
	for key := range jobsMap {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return lessConfigKey(keys[i], keys[j]) })

	for _, key := range keys {
		jobs = append(jobs, jobsMap[key])
	}

	return jobs
}

// resolveCronJobs substitutes the placeholders each job names and drops the
// ones that cannot be resolved, returning what may be installed and a line
// about each refusal.
//
// Dropping rather than installing verbatim: a line with `{{workdir}}` still in
// it is a job that runs on schedule and fails on schedule, into /dev/null.
func resolveCronJobs(jobs []string, projectConf map[string]string) (resolved []string, refusals []string) {
	for _, job := range jobs {
		expanded, unresolved := expandCronJob(job, projectConf)
		if len(unresolved) > 0 {
			refusals = append(refusals, job+"\n    unresolved: "+strings.Join(unresolved, ", "))
			continue
		}
		resolved = append(resolved, expanded)
	}
	return resolved, refusals
}

// lessConfigKey orders two "/"-separated config keys, comparing a segment as a
// number when both sides are entirely digits.
func lessConfigKey(a, b string) bool {
	as, bs := strings.Split(a, "/"), strings.Split(b, "/")
	for i := 0; i < len(as) && i < len(bs); i++ {
		if as[i] == bs[i] {
			continue
		}
		an, aErr := strconv.Atoi(as[i])
		bn, bErr := strconv.Atoi(bs[i])
		if aErr == nil && bErr == nil {
			return an < bn
		}
		return as[i] < bs[i]
	}
	return len(as) < len(bs)
}

// platformInstallsOwnCron reports whether CronExecute has a platform branch that
// writes a crontab of its own. For everything else the config is the only source
// of jobs, so an empty job list means nothing is scheduled at all.
func platformInstallsOwnCron(platform string) bool {
	switch platform {
	case "magento2", "shopify", "shopware":
		return true
	}
	return false
}

// CronJobCount reports how many jobs are installed in the container's crontab,
// and whether the question could be answered at all.
//
// This is the half `service cron status` cannot see. The daemon being up says a
// process is running; it says nothing about there being anything for it to run,
// and those two answers came apart on a live project for four days without a
// single line of output anywhere.
//
// A container that is not there answers (0, false) — unknown, which is not the
// same as none and must not be printed as though it were.
func CronJobCount(projectName string) (int, bool) {
	projectConf := configs2.GetProjectConfig(projectName)
	service := resolveMainService(projectConf)
	service, userOS, _ := cliHelper.GetEnvForUserServiceWorkdir(service, "root", "")

	out, err := containerExecSilent(GetContainerName(projectConf, projectName, service), userOS, "crontab", "-u", "www-data", "-l")
	if err != nil {
		// `crontab -l` exits non-zero when the user simply has no crontab. That
		// is an answer — none — and only anything else is a failure to ask.
		if strings.Contains(out, "no crontab for") {
			return 0, true
		}
		return 0, false
	}

	count := 0
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		count++
	}
	return count, true
}

// installCronJobsFromConfig installs cron jobs from configuration
func installCronJobsFromConfig(projectConf map[string]string, projectName string, manual bool) {
	jobs, refusals := resolveCronJobs(getCronJobsFromConfig(projectConf), projectConf)
	for _, refusal := range refusals {
		// Always, not only under `manual`: a job that never reaches the crontab
		// is invisible everywhere else, which is the failure this whole area
		// keeps producing.
		fmtc.WarningLn("Cron job not installed — " + refusal)
	}
	if len(jobs) == 0 {
		// An empty list is an instruction, not an absence of one: the config owns
		// its block, and leaving the previous jobs in place means a job deleted
		// from the config goes on running forever with nothing naming it. Only
		// that block goes — Magento's and anything added by hand stay.
		removeCronJobsFromConfig(projectConf, projectName, false)
		if manual {
			fmt.Println("No cron jobs defined in configuration")
		} else if !platformInstallsOwnCron(projectConf["platform"]) {
			// Auto-invocation, from start or rebuild. The cron daemon has just
			// been started and there is nothing for it to run, and on these
			// platforms nothing else will add anything later — so `status` will
			// go on reporting a scheduler that schedules nothing. Said once,
			// where the start output is read.
			fmtc.WarningLn("Cron is started but no jobs are defined in cron/jobs — nothing will run on a schedule.")
		}
		return
	}

	containerName := GetContainerName(projectConf, projectName, resolveMainService(projectConf))

	// Read, merge, write: the jobs go into a block of their own and everything
	// else in the crontab is carried over untouched.
	err := ContainerExec(containerName, "root", false, "bash", "-c",
		writeCrontabScript(mergeCrontab(readCrontab(projectConf, projectName), jobs)))

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

	err := ContainerExec(containerName, "root", false, "bash", "-c",
		writeCrontabScript(removeMadockBlock(readCrontab(projectConf, projectName))))

	if manual {
		if err != nil {
			logger.Println(err)
			fmt.Println("Cron jobs removed (or none existed)")
		} else {
			fmtc.SuccessLn("Cron jobs removed")
		}
	}
}

// readCrontab returns www-data's current crontab, or an empty string when there
// is none or the container cannot be reached.
//
// An empty string is safe for both callers: the merge then writes only our own
// block, which is what a fresh container needs anyway. Getting it wrong in the
// other direction would not be — a failed read treated as "the crontab is
// something else" would carry stale text back in.
func readCrontab(projectConf map[string]string, projectName string) string {
	service := resolveMainService(projectConf)
	service, userOS, _ := cliHelper.GetEnvForUserServiceWorkdir(service, "root", "")

	out, err := containerExecSilent(GetContainerName(projectConf, projectName, service), userOS, "crontab", "-u", "www-data", "-l")
	if err != nil {
		return ""
	}
	return out
}
