package n98

import (
	cliHelper "github.com/faradey/madock/src/helper/cli"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	"github.com/faradey/madock/src/helper/configs"
	"log"
	"os"
	"os/exec"
	"strings"
)

func Execute() {
	flag := cliHelper.NormalizeCliCommandWithJoin(os.Args[2:])
	projectName := configs.GetProjectName()
	projectConf := configs.GetCurrentProjectConfig()

	if projectConf["PLATFORM"] == "magento2" {
		cmd := exec.Command("docker", "exec", "-it", "-u", "www-data", strings.ToLower(projectConf["CONTAINER_NAME_PREFIX"])+strings.ToLower(projectName)+"-php-1", "bash", "-c", "cd "+projectConf["WORKDIR"]+" && /var/www/n98magerun/n98-magerun2.phar "+flag)
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
