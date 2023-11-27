package magento

import (
	"github.com/faradey/madock/src/configs"
	cliHelper "github.com/faradey/madock/src/helper/cli"
	"log"
	"os"
	"os/exec"
	"strings"
)

func Execute() {
	flag := cliHelper.NormalizeCliCommandWithJoin(os.Args[2:])
	projectName := configs.GetProjectName()
	projectConf := configs.GetCurrentProjectConfig()
	cmd := exec.Command("docker", "exec", "-it", "-u", "www-data", strings.ToLower(projectConf["CONTAINER_NAME_PREFIX"])+strings.ToLower(projectName)+"-php-1", "bash", "-c", "cd "+projectConf["WORKDIR"]+" && php bin/magento "+flag)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}
