package shopify

import (
	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/helper/cli"
	"log"
	"os"
	"os/exec"
	"strings"
)

func Execute() {
	flag := cli.NormalizeCliCommandWithJoin(os.Args[2:])
	projectName := configs.GetProjectName()
	projectConf := configs.GetCurrentProjectConfig()
	service, user, workdir := cli.GetEnvForUserServiceWorkdir("php", "www-data", projectConf["WORKDIR"])
	cmd := exec.Command("docker", "exec", "-it", "-u", user, strings.ToLower(projectConf["CONTAINER_NAME_PREFIX"])+strings.ToLower(projectName)+"-"+service+"-1", "bash", "-c", "cd "+workdir+" && "+flag)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}
