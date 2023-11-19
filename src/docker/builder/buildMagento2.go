package builder

import (
	"github.com/faradey/madock/src/cli/attr"
	"github.com/faradey/madock/src/cli/fmtc"
	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/controller/general/cron"
	"github.com/faradey/madock/src/paths"
	"log"
	"os"
	"os/exec"
	"os/user"
	"strings"
)

func StartMagento2(withChown bool, projectConf map[string]string) {
	projectName := configs.GetProjectName()
	UpNginx()
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
		"start",
	}
	cmd := exec.Command("docker", profilesOn...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmtc.ToDoLn("Creating containers")
		upProjectWithBuild(attr.Options.WithChown)
	} else {
		if withChown {
			projectName := configs.GetProjectName()
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

		if val, ok := projectConf["CRON_ENABLED"]; ok && val == "true" {
			cron.RunCron(true, false)
		} else {
			cron.RunCron(false, false)
		}
	}
}

func StopMagento2() {
	projectName := configs.GetProjectName()
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
		"stop",
	}
	cmd := exec.Command("docker", profilesOn...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func MftfInit() {
	projectName := configs.GetProjectName()
	projectConf := configs.GetCurrentProjectConfig()

	cmd := exec.Command("docker", "exec", "-it", "-u", "root", strings.ToLower(projectConf["CONTAINER_NAME_PREFIX"])+strings.ToLower(projectName)+"-php-1", "bash", "-c", "cd /var/www/html && bin/magento config:set cms/wysiwyg/enabled disabled && bin/magento config:set admin/security/admin_account_sharing 1 && bin/magento config:set admin/security/use_form_key 0 && bin/magento config:set web/seo/use_rewrites 1 && bin/magento config:set twofactorauth/general/force_providers google && bin/magento config:set twofactorauth/google/otp_window 60 && bin/magento security:tfa:google:set-secret "+projectConf["MFTF_ADMIN_USER"]+" "+projectConf["MFTF_OTP_SHARED_SECRET"]+" && bin/magento cache:clean config full_page")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}
