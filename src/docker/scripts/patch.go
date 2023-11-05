package scripts

import (
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/faradey/madock/src/configs"
)

func CreatePatch(filePath, patchName, title string, force bool) {
	containerName := "php"
	projectName := configs.GetProjectName()
	projectConfig := configs.GetCurrentProjectConfig()
	isForce := ""
	if force {
		isForce = "f"
	}
	cmd := exec.Command("docker", "exec", "-it", "-u", "www-data", strings.ToLower(projectConfig["CONTAINER_NAME_PREFIX"])+strings.ToLower(projectName)+"-"+containerName+"-1", "php", "/var/www/scripts/php/patch-create.php", projectConfig["WORKDIR"], filePath, patchName, title, isForce)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}
