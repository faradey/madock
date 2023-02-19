package scripts

import (
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/faradey/madock/src/paths"
)

func MagentoInfo() {
	containerName := "php"
	projectName := paths.GetProjectName()
	cmd := exec.Command("docker", "exec", "-it", strings.ToLower(projectName)+"-"+containerName+"-1", "php", "/var/www/scripts/php/magento-info.php")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}
