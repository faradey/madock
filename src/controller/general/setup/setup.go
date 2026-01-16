package setup

import (
	"fmt"
	"strings"

	setupCustom "github.com/faradey/madock/src/controller/custom/setup"
	setupMagento "github.com/faradey/madock/src/controller/magento/setup"
	setupPrestashop "github.com/faradey/madock/src/controller/prestashop/setup"
	setupPWA "github.com/faradey/madock/src/controller/pwa/setup"
	setupShopify "github.com/faradey/madock/src/controller/shopify/setup"
	setupShopware "github.com/faradey/madock/src/controller/shopware/setup"
	"github.com/faradey/madock/src/helper/cli/arg_struct"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	configs2 "github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/detect"
	"github.com/faradey/madock/src/helper/logger"
	"github.com/faradey/madock/src/helper/paths"
	"github.com/faradey/madock/src/helper/setup/tools"
)

func Execute() {
	args := attr.Parse(new(arg_struct.ControllerGeneralSetup)).(*arg_struct.ControllerGeneralSetup)

	// Display setup banner
	fmt.Println("")
	fmtc.Banner("MADOCK SETUP", "Docker Environment Configuration")
	fmt.Println("")

	projectName := configs2.GetProjectName()
	hasConfig := configs2.IsHasConfig(projectName)
	continueSetup := true
	if hasConfig {
		fmtc.WarningLn("File config.xml is already exist in project " + projectName)
		if args.Yes {
			// Auto-confirm with --yes flag
			continueSetup = true
		} else {
			if !fmtc.Confirm("Do you want to continue?", false) {
				if !args.Download && !args.Install {
					logger.Fatal("Exit")
				}
				continueSetup = false
			}
		}
	}

	if strings.Contains(projectName, ".") || strings.Contains(projectName, " ") {
		fmtc.ErrorLn("The project folder name cannot contain a period or space")
		return
	}

	envFile := paths.MakeDirsByPath(paths.GetExecDirPath()+"/projects/"+projectName) + "/config.xml"
	var projectConf map[string]string
	if paths.IsFileExist(envFile) {
		projectConf = configs2.GetProjectConfig(projectName)
	} else {
		projectConf = configs2.GetGeneralConfig()
	}

	platform := args.Platform
	detectedVersion := ""

	// Try to auto-detect platform from composer.json
	if platform == "" {
		detection := detect.Detect(paths.GetRunDirPath())
		if detection.Detected {
			fmtc.SuccessIconLn(fmt.Sprintf("Detected: %s %s", detection.Platform, detection.PlatformVersion))
			fmtc.PrintKeyValue("Source", detection.Source)
			fmt.Println("")

			// Auto-confirm with --yes flag
			if args.Yes || fmtc.Confirm("Use detected configuration?", true) {
				platform = detection.Platform
				detectedVersion = detection.PlatformVersion
			}
			fmt.Println("")
		}
	}

	if platform == "" {
		platform = tools.Platform()
	}

	if platform == "magento2" {
		setupMagento.ExecuteWithVersion(projectName, projectConf, continueSetup, args, detectedVersion)
	} else if platform == "pwa" {
		setupPWA.Execute(projectName, projectConf, continueSetup, args)
	} else if platform == "shopify" {
		setupShopify.Execute(projectName, projectConf, continueSetup, args)
	} else if platform == "custom" {
		setupCustom.Execute(projectName, projectConf, continueSetup, args)
	} else if platform == "shopware" {
		setupShopware.Execute(projectName, projectConf, continueSetup, args)
	} else if platform == "prestashop" {
		setupPrestashop.Execute(projectName, projectConf, continueSetup, args)
	}
}
