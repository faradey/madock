package bash

import (
	"os"
	"os/exec"
	"strings"

	"github.com/faradey/madock/src/helper/cli/arg_struct"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/docker"
	"github.com/faradey/madock/src/helper/logger"
)

var allowedShells = map[string]bool{
	"bash": true,
	"sh":   true,
	"zsh":  true,
	"ash":  true,
	"fish": true,
}

func Execute() {
	args := attr.Parse(new(arg_struct.ControllerGeneralBash)).(*arg_struct.ControllerGeneralBash)

	service := "php"
	user := "root"
	projectConf := configs.GetCurrentProjectConfig()
	if projectConf["platform"] == "pwa" {
		service = "nodejs"
	}

	if args.Service != "" {
		service = args.Service
	}

	if args.User != "" {
		user = args.User
	}

	projectName := configs.GetProjectName()
	shell := "bash"
	if args.Shell != "" {
		shell = strings.TrimSpace(args.Shell)
		if !allowedShells[shell] {
			fmtc.ErrorLn("Invalid shell. Allowed shells: bash, sh, zsh, ash, fish")
			return
		}
	}
	cmd := exec.Command("docker", "exec", "-it", "-u", user, docker.GetContainerName(projectConf, projectName, service), shell)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		logger.Fatal(err)
	}
}
