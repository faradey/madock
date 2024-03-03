package cloud

import (
	cliHelper "github.com/faradey/madock/src/helper/cli"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/docker"
	"github.com/faradey/madock/src/helper/logger"
	"os"
	"os/exec"
	"strings"
)

type ArgsStruct struct {
	attr.ArgumentsWithArgs
}

func Cloud() {
	attr.Parse(new(ArgsStruct))

	projectConf := configs.GetCurrentProjectConfig()
	if projectConf["platform"] == "magento2" {
		flag := cliHelper.NormalizeCliCommandWithJoin(os.Args[2:])
		flag = strings.Replace(flag, "$project", projectConf["magento/cloud/project_name"], -1)

		projectName := configs.GetProjectName()
		cmd := exec.Command("docker", "exec", "-it", "-u", "www-data", docker.GetContainerName(projectConf, projectName, "php"), "bash", "-c", "cd "+projectConf["workdir"]+" && magento-cloud "+flag)
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
