package mftf

import (
	cliHelper "github.com/faradey/madock/src/helper/cli"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/docker"
	"log"
	"os"
	"os/exec"
)

func Init() {
	projectName := configs.GetProjectName()
	projectConf := configs.GetCurrentProjectConfig()

	if projectConf["PLATFORM"] == "magento2" {
		cmd := exec.Command("docker", "exec", "-it", "-u", "root", docker.GetContainerName(projectConf, projectName, "php"), "bash", "-c", "cd "+projectConf["WORKDIR"]+" && bin/magento config:set cms/wysiwyg/enabled disabled && bin/magento config:set admin/security/admin_account_sharing 1 && bin/magento config:set admin/security/use_form_key 0 && bin/magento config:set web/seo/use_rewrites 1 && bin/magento config:set twofactorauth/general/force_providers google && bin/magento config:set twofactorauth/google/otp_window 60 && bin/magento security:tfa:google:set-secret "+projectConf["MFTF_ADMIN_USER"]+" "+projectConf["MFTF_OTP_SHARED_SECRET"]+" && bin/magento cache:clean config full_page")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			log.Fatal(err)
		}
	} else {
		fmtc.Warning("This command is not supported for " + projectConf["PLATFORM"])
	}
}

func Execute() {
	flag := cliHelper.NormalizeCliCommandWithJoin(os.Args[2:])
	projectName := configs.GetProjectName()
	projectConf := configs.GetCurrentProjectConfig()

	if projectConf["PLATFORM"] == "magento2" {
		cmd := exec.Command("docker", "exec", "-it", "-u", "www-data", docker.GetContainerName(projectConf, projectName, "php"), "bash", "-c", "cd "+projectConf["WORKDIR"]+" && php vendor/bin/mftf "+flag)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			log.Fatal(err)
		}
	} else {
		fmtc.Warning("This command is not supported for " + projectConf["PLATFORM"])
	}
}
