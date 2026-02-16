package mftf

import (
	"os"
	"os/exec"

	"github.com/faradey/madock/src/command"
	cliHelper "github.com/faradey/madock/src/helper/cli"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/docker"
	"github.com/faradey/madock/src/helper/logger"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"mftf"},
		Handler:  Execute,
		Help:     "Execute MFTF",
		Category: "magento",
	})
	command.Register(&command.Definition{
		Aliases:  []string{"mftf:init"},
		Handler:  Init,
		Help:     "Initialize MFTF",
		Category: "magento",
	})
}

func Init() {
	projectName := configs.GetProjectName()
	projectConf := configs.GetCurrentProjectConfig()

	if projectConf["platform"] == "magento2" {
		cmd := exec.Command("docker", "exec", "-it", "-u", "root", docker.GetContainerName(projectConf, projectName, "php"), "bash", "-c", "cd "+projectConf["workdir"]+" && bin/magento config:set cms/wysiwyg/enabled disabled && bin/magento config:set admin/security/admin_account_sharing 1 && bin/magento config:set admin/security/use_form_key 0 && bin/magento config:set web/seo/use_rewrites 1 && bin/magento config:set twofactorauth/general/force_providers google && bin/magento config:set twofactorauth/google/otp_window 60 && bin/magento security:tfa:google:set-secret "+projectConf["magento/mftf/admin_user"]+" "+projectConf["magento/mftf/otp_shared_secret"]+" && bin/magento cache:clean config full_page")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			logger.Fatal(err)
		}
	} else {
		fmtc.Warning("This command is not supported for " + projectConf["platform"])
	}
}

func Execute() {
	flag := cliHelper.NormalizeCliCommandWithJoin(os.Args[2:])
	projectName := configs.GetProjectName()
	projectConf := configs.GetCurrentProjectConfig()

	if projectConf["platform"] == "magento2" {
		cmd := exec.Command("docker", "exec", "-it", "-u", "www-data", docker.GetContainerName(projectConf, projectName, "php"), "bash", "-c", "cd "+projectConf["workdir"]+" && php vendor/bin/mftf "+flag)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			logger.Fatal(err)
		}
	} else {
		fmtc.Warning("This command is not supported for " + projectConf["platform"])
	}
}
