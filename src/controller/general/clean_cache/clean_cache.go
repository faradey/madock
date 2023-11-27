package clean_cache

import (
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/jessevdk/go-flags"
	"log"
	"os"
	"os/exec"
	"strings"
)

type ArgsStruct struct {
	attr.Arguments
	User string `long:"user" short:"u" description:"User"`
}

func CleanCache() {
	args := getArgs()

	user := "www-data"

	if args.User != "" {
		user = args.User
	}

	projectName := configs.GetProjectName()
	projectConf := configs.GetCurrentProjectConfig()

	if projectConf["PLATFORM"] == "magento2" {
		cmd := exec.Command("docker", "exec", "-it", "-u", user, strings.ToLower(projectConf["CONTAINER_NAME_PREFIX"])+strings.ToLower(projectName)+"-php-1", "bash", "-c", "cd "+projectConf["WORKDIR"]+" && rm -f pub/static/deployed_version.txt && rm -Rf pub/static/frontend && rm -Rf pub/static/adminhtml && rm -Rf var/view_preprocessed/pub && rm -Rf generated/code && php bin/magento c:f")
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
