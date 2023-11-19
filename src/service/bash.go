package service

import (
	"github.com/faradey/madock/src/configs"
	"log"
	"os"
	"os/exec"
	"strings"
)

func Bash(containerName, user string) {
	projectName := configs.GetProjectName()
	projectConfig := configs.GetCurrentProjectConfig()
	cmd := exec.Command("docker", "exec", "-it", "-u", user, strings.ToLower(projectConfig["CONTAINER_NAME_PREFIX"])+strings.ToLower(projectName)+"-"+containerName+"-1", "bash")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}
