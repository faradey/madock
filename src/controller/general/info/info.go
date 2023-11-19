package info

import (
	"github.com/faradey/madock/src/cli/attr"
	"github.com/faradey/madock/src/cli/fmtc"
	"github.com/faradey/madock/src/configs"
	"github.com/jessevdk/go-flags"
	"log"
	"os"
	"os/exec"
	"strings"
)

type ArgsStruct struct {
	attr.Arguments
}

func Info() {
	getArgs()

	service := "php"
	projectConf := configs.GetCurrentProjectConfig()
	if projectConf["PLATFORM"] == "pwa" {
		service = "nodejs"
	}

	if projectConf["PLATFORM"] == "magento2" {
		projectName := configs.GetProjectName()
		cmd := exec.Command("docker", "exec", "-it", strings.ToLower(projectConf["CONTAINER_NAME_PREFIX"])+strings.ToLower(projectName)+"-"+service+"-1", "php", "/var/www/scripts/php/magento-info.php", projectConf["WORKDIR"])
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

func getArgs() *ArgsStruct {
	args := new(ArgsStruct)
	if len(os.Args) > 2 {
		argsOrigin := os.Args[2:]
		var err error
		_, err = flags.ParseArgs(args, argsOrigin)

		if err != nil {
			log.Fatal(err)
		}
	}

	return args
}
