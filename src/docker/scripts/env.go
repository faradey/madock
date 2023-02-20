package scripts

import (
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/faradey/madock/src/configs"
)

func CreateEnv(conf, host string) {
	containerName := "php"
	projectName := configs.GetProjectName()
	cmd := exec.Command("docker", "exec", "-it", "-u", "www-data", strings.ToLower(projectName)+"-"+containerName+"-1", "php", "/var/www/scripts/php/env-create.php", conf, host)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}
