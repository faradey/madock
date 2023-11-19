package cli

import (
	"github.com/faradey/madock/src/configs"
	cliHelper "github.com/faradey/madock/src/helper"
	"log"
	"os"
	"os/exec"
	"strings"
)

func Cli() {
	flag := cliHelper.NormalizeCliCommandWithJoin(os.Args[2:])
	projectName := configs.GetProjectName()
	projectConfig := configs.GetCurrentProjectConfig()
	containerName := "php"
	if projectConfig["PLATFORM"] == "pwa" {
		containerName = "nodejs"
	}

	if os.Getenv("MADOCK_SERVICE_NAME") != "" {
		containerName = os.Getenv("MADOCK_SERVICE_NAME")
	}

	workdir := ""

	if os.Getenv("MADOCK_WORKDIR") != "" {
		workdir = "cd " + os.Getenv("MADOCK_WORKDIR") + " && "
	}

	user := "www-data"

	if os.Getenv("MADOCK_USER") != "" {
		user = os.Getenv("MADOCK_USER")
	}

	cmd := exec.Command("docker", "exec", "-it", "-u", user, strings.ToLower(projectConfig["CONTAINER_NAME_PREFIX"])+strings.ToLower(projectName)+"-"+containerName+"-1", "bash", "-c", workdir+flag)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}
