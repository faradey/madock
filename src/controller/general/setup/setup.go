package setup

import (
	"bufio"
	"fmt"
	"github.com/alexflint/go-arg"
	setupCustom "github.com/faradey/madock/src/controller/custom/setup"
	setupMagento "github.com/faradey/madock/src/controller/magento/setup"
	setupPWA "github.com/faradey/madock/src/controller/pwa/setup"
	setupShopify "github.com/faradey/madock/src/controller/shopify/setup"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	configs2 "github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/paths"
	"github.com/faradey/madock/src/helper/setup/tools"
	"log"
	"os"
	"strings"
)

type ArgsStruct struct {
	attr.Arguments
	Download   bool `arg:"-d,--download" help:"Download code from repository"`
	Install    bool `arg:"-i,--install" help:"Install service (Magento, PWA, Shopify SDK, etc.)"`
	SampleData bool `arg:"-s,--sample-data" help:"Sample data"`
}

func Execute() {
	args := getArgs()

	projectName := configs2.GetProjectName()
	hasConfig := configs2.IsHasConfig(projectName)
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
	if paths.IsFileExist(envFile) {
		projectConf = configs2.GetProjectConfig(projectName)
	} else {
		projectConf = configs2.GetGeneralConfig()
	}

	fmt.Println("")
	fmtc.Title("Specify Platform: ")
	platform := tools.Platform()
	if platform == "magento2" {
		setupMagento.Execute(projectName, projectConf, continueSetup, args.Download, args.Install, args.SampleData)
	} else if platform == "pwa" {
		setupPWA.Execute(projectName, projectConf, continueSetup)
	} else if platform == "shopify" {
		setupShopify.Execute(projectName, projectConf, continueSetup)
	} else if platform == "custom" {
		setupCustom.Execute(projectName, projectConf, continueSetup)
	}
}

func getArgs() *ArgsStruct {
	args := new(ArgsStruct)
	if attr.IsParseArgs && len(os.Args) > 2 {
		argsOrigin := os.Args[2:]
		p, err := arg.NewParser(arg.Config{
			IgnoreEnv: true,
		}, args)

		if err != nil {
			log.Fatal(err)
		}

		err = p.Parse(argsOrigin)

		if err != nil {
			log.Fatal(err)
		}
	}

	attr.IsParseArgs = false
	return args
}
