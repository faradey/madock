package cloud

import (
	"github.com/faradey/madock/src/cli/fmtc"
	"github.com/faradey/madock/src/configs"
	cliHelper "github.com/faradey/madock/src/helper"
	"log"
	"os"
	"os/exec"
	"strings"
)

func Cloud() {
	projectConf := configs.GetCurrentProjectConfig()
	if projectConf["PLATFORM"] == "magento2" {
		flag := cliHelper.NormalizeCliCommandWithJoin(os.Args[2:])
		flag = strings.Replace(flag, "$project", projectConf["MAGENTOCLOUD_PROJECT_NAME"], -1)

		projectName := configs.GetProjectName()
		cmd := exec.Command("docker", "exec", "-it", "-u", "www-data", strings.ToLower(projectConf["CONTAINER_NAME_PREFIX"])+strings.ToLower(projectName)+"-php-1", "bash", "-c", "cd "+projectConf["WORKDIR"]+" && magento-cloud "+flag)
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
