package scripts

import (
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/faradey/madock/src/configs"
)

func MagentoInfo() {
	containerName := "php"
	projectName := configs.GetProjectName()
	projectConfig := configs.GetCurrentProjectConfig()
	cmd := exec.Command("docker", "exec", "-it", strings.ToLower(projectConfig["CONTAINER_NAME_PREFIX"])+"_"+strings.ToLower(projectName)+"-"+containerName+"-1", "php", "/var/www/scripts/php/magento-info.php", projectConfig["WORKDIR"])
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}
