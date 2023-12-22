package clean_cache

import (
	"fmt"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/docker"
	"github.com/jessevdk/go-flags"
	"log"
	"os"
	"os/exec"
	"sync"
)

type ArgsStruct struct {
	attr.Arguments
	User string `long:"user" short:"u" description:"User"`
}

func Execute() {
	args := getArgs()

	user := "www-data"

	if args.User != "" {
		user = args.User
	}

	projectName := configs.GetProjectName()
	projectConf := configs.GetCurrentProjectConfig()

	if projectConf["PLATFORM"] == "magento2" {
		commands := []string{"rm -f pub/static/deployed_version.txt", "rm -Rf pub/static/frontend", "rm -Rf pub/static/adminhtml", "rm -Rf var/view_preprocessed/pub", "rm -Rf generated/code"}
		cmd := exec.Command("docker", "exec", "-it", "-u", user, docker.GetContainerName(projectConf, projectName, "php"), "bash", "-c", "cd "+projectConf["WORKDIR"]+" && "+"php bin/magento c:f")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			log.Fatal(err)
		}

		var waitGroup sync.WaitGroup
		for _, command := range commands {
			waitGroup.Add(1)
			command := command
			go func() {
				defer waitGroup.Done()
				cmd := exec.Command("docker", "exec", "-it", "-u", user, docker.GetContainerName(projectConf, projectName, "php"), "bash", "-c", "cd "+projectConf["WORKDIR"]+" && "+command)
				cmd.Stdin = os.Stdin
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				err := cmd.Run()
				if err != nil {
					fmt.Println("Error: " + err.Error())
				}
			}()
		}
		waitGroup.Wait()
		fmt.Println("Cache cleared successfully")
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
