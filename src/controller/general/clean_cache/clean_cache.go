package clean_cache

import (
	"fmt"
	"os"
	"os/exec"
	"sync"

	"github.com/faradey/madock/src/command"
	"github.com/faradey/madock/src/helper/cli/arg_struct"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/docker"
	"github.com/faradey/madock/src/helper/logger"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"c:f"},
		Handler:  Execute,
		Help:     "Flush cache",
		Category: "general",
	})
}

func Execute() {
	args := attr.Parse(new(arg_struct.ControllerGeneralCleanCache)).(*arg_struct.ControllerGeneralCleanCache)

	user := "www-data"

	if args.User != "" {
		user = args.User
	}

	projectName := configs.GetProjectName()
	projectConf := configs.GetCurrentProjectConfig()

	if projectConf["platform"] == "magento2" {
		commands := []string{"rm -f pub/static/deployed_version.txt", "rm -Rf pub/static/frontend", "rm -Rf pub/static/adminhtml", "rm -Rf var/view_preprocessed/pub", "rm -Rf generated/code"}
		cmd := exec.Command("docker", "exec", "-it", "-u", user, docker.GetContainerName(projectConf, projectName, "php"), "bash", "-c", "cd "+projectConf["workdir"]+" && "+"php bin/magento c:f")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			logger.Fatal(err)
		}

		var waitGroup sync.WaitGroup
		for _, command := range commands {
			waitGroup.Add(1)
			command := command
			go func() {
				defer waitGroup.Done()
				cmd := exec.Command("docker", "exec", "-it", "-u", user, docker.GetContainerName(projectConf, projectName, "php"), "bash", "-c", "cd "+projectConf["workdir"]+" && "+command)
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
		fmtc.Warning("This command is not supported for " + projectConf["platform"])
	}
}
