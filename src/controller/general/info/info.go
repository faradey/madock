package info

import (
	"github.com/alexflint/go-arg"
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
	getArgs()

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

func getArgs() *ArgsStruct {
	args := new(ArgsStruct)
	if attr.IsParseArgs && len(os.Args) > 2 {
		argsOrigin := os.Args[2:]
		p, err := arg.NewParser(arg.Config{
			IgnoreEnv: true,
		}, args)

		if err != nil {
			log.Fatal(err)
		}

		err = p.Parse(argsOrigin)

		if err != nil {
			log.Fatal(err)
		}
	}

	attr.IsParseArgs = false
	return args
}
