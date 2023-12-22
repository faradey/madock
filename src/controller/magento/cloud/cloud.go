package cloud

import (
	cliHelper "github.com/faradey/madock/src/helper/cli"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/docker"
	"github.com/jessevdk/go-flags"
	"log"
	"os"
	"os/exec"
	"strings"
)

type ArgsStruct struct {
	attr.Arguments
}

func Cloud() {
	getArgs()

	projectConf := configs.GetCurrentProjectConfig()
	if projectConf["PLATFORM"] == "magento2" {
		flag := cliHelper.NormalizeCliCommandWithJoin(os.Args[2:])
		flag = strings.Replace(flag, "$project", projectConf["MAGENTOCLOUD_PROJECT_NAME"], -1)

		projectName := configs.GetProjectName()
		cmd := exec.Command("docker", "exec", "-it", "-u", "www-data", docker.GetContainerName(projectConf, projectName, "php"), "bash", "-c", "cd "+projectConf["WORKDIR"]+" && magento-cloud "+flag)
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
	if attr.IsParseArgs && len(os.Args) > 2 {
		argsOrigin := os.Args[2:]
		var err error
		_, err = flags.ParseArgs(args, argsOrigin)

		if err != nil {
			log.Fatal(err)
		}
	}

	attr.IsParseArgs = false
	return args
}
