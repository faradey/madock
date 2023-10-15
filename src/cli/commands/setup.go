package commands

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/faradey/madock/src/cli/commands/setup"
	"github.com/faradey/madock/src/docker/scripts"
	"log"
	"os"
	"strings"

	"github.com/faradey/madock/src/cli/attr"
	"github.com/faradey/madock/src/cli/fmtc"
	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/paths"
)

func Setup() {
	projectName := configs.GetProjectName()
	hasConfig := configs.IsHasConfig(projectName)
	continueSetup := true
	if hasConfig {
		fmtc.WarningLn("File env is already exist in project " + projectName)
		fmt.Println("Do you want to continue? (y/N)")
		fmt.Print("> ")

		buf := bufio.NewReader(os.Stdin)
		sentence, err := buf.ReadBytes('\n')
		selected := strings.TrimSpace(string(sentence))
		if err != nil {
			log.Fatal(err)
		} else if selected != "y" {
			if !attr.Options.Download && !attr.Options.Install {
				log.Fatal("Exit")
			}
			continueSetup = false
		}
	}

	if strings.Contains(projectName, ".") || strings.Contains(projectName, " ") {
		fmtc.ErrorLn("The project folder name cannot contain a period or space")
		return
	}

	fmtc.SuccessLn("Start set up environment")

	envFile := paths.MakeDirsByPath(paths.GetExecDirPath()+"/projects/"+projectName) + "/env.txt"
	var projectConfig map[string]string
	if _, err := os.Stat(envFile); !os.IsNotExist(err) {
		projectConfig = configs.GetProjectConfig(projectName)
	} else {
		projectConfig = configs.GetGeneralConfig()
	}

	fmt.Println("")
	fmtc.Title("Specify Platform: ")
	platform := setup.Platform()
	if platform == "magento2" {
		setup.Magento2(projectName, projectConfig, continueSetup)
	} else if platform == "pwa" {
		setup.PWA(projectName, projectConfig, continueSetup)
	} else if platform == "shopify" {
		setup.Shopify(projectName, projectConfig, continueSetup)
	}
}

func SetupEnv() {
	envFile := paths.GetRunDirPath() + "/app/etc/env.php"
	if _, err := os.Stat(envFile); !os.IsNotExist(err) && !attr.Options.Force {
		log.Fatal("The env.php file is already exist.")
	} else {
		data, err := json.Marshal(configs.GetCurrentProjectConfig())
		if err != nil {
			log.Fatal(err)
		}
		scripts.CreateEnv(string(data), attr.Options.Host)
	}
}
