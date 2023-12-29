package info

import (
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/docker"
	"log"
	"os"
	"os/exec"
)

type ArgsStruct struct {
	attr.Arguments
}

func Info() {
	attr.Parse(new(ArgsStruct))

	service := "php"
	projectConf := configs.GetCurrentProjectConfig()
	if projectConf["platform"] == "pwa" {
		service = "nodejs"
	}

	if projectConf["platform"] == "magento2" {
		projectName := configs.GetProjectName()
		cmd := exec.Command("docker", "exec", "-it", docker.GetContainerName(projectConf, projectName, service), "php", "/var/www/scripts/php/magento-info.php", projectConf["workdir"])
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			log.Fatal(err)
		}
	} else {
		fmtc.Warning("This command is not supported for " + projectConf["platform"])
	}
}
