package setup

import (
	"bufio"
	"fmt"
	"github.com/faradey/madock/src/cli/attr"
	"github.com/faradey/madock/src/cli/fmtc"
	"github.com/faradey/madock/src/configs"
	setupCustom "github.com/faradey/madock/src/controller/custom/setup"
	setupMagento "github.com/faradey/madock/src/controller/magento/setup"
	setupPWA "github.com/faradey/madock/src/controller/pwa/setup"
	setupShopify "github.com/faradey/madock/src/controller/shopify/setup"
	"github.com/faradey/madock/src/helper/paths"
	"github.com/faradey/madock/src/helper/setup/tools"
	"github.com/jessevdk/go-flags"
	"log"
	"os"
	"strings"
)

type ArgsStruct struct {
	attr.Arguments
	Download    bool `long:"download" short:"d" description:"Download code from repository"`
	Install     bool `long:"install" short:"i" description:"Install service (Magento, PWA, Shopify SDK, etc.)"`
	SampleData  bool `long:"sample-data" short:"s" description:"sample-data"`
	WithChown   bool `long:"with-chown" short:"c" description:"With Chown"`
	WithVolumes bool `long:"with-volumes" description:"With Volumes"`
}

func Execute() {
	args := getArgs()

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
			if !args.Download && !args.Install {
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
	var projectConf map[string]string
	if _, err := os.Stat(envFile); !os.IsNotExist(err) {
		projectConf = configs.GetProjectConfig(projectName)
	} else {
		projectConf = configs.GetGeneralConfig()
	}

	fmt.Println("")
	fmtc.Title("Specify Platform: ")
	platform := tools.Platform()
	if platform == "magento2" {
		setupMagento.Execute(projectName, projectConf, continueSetup, args.Download, args.Install, args.WithChown, args.WithVolumes, args.SampleData)
	} else if platform == "pwa" {
		setupPWA.Execute(projectName, projectConf, continueSetup, args.WithChown, args.WithVolumes)
	} else if platform == "shopify" {
		setupShopify.Execute(projectName, projectConf, continueSetup, args.WithChown, args.WithVolumes)
	} else if platform == "custom" {
		setupCustom.Execute(projectName, projectConf, continueSetup, args.WithChown, args.WithVolumes)
	}
}

func getArgs() *ArgsStruct {
	args := new(ArgsStruct)
	if len(os.Args) > 2 {
		argsOrigin := os.Args[2:]
		var err error
		_, err = flags.ParseArgs(args, argsOrigin)

		if err != nil {
			log.Fatal(err)
		}
	}

	return args
}
