package setup

import (
	"fmt"
	"strings"

	"github.com/faradey/madock/src/command"
	"github.com/faradey/madock/src/helper/cli/arg_struct"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	configs2 "github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/detect"
	"github.com/faradey/madock/src/helper/logger"
	"github.com/faradey/madock/src/helper/paths"
	"github.com/faradey/madock/src/helper/setup/tools"
	setupreg "github.com/faradey/madock/src/setup"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"setup"},
		Handler:  Execute,
		Help:     "Setup project",
		Category: "setup",
	})
}

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

	detectedLanguage := ""

	// Try to auto-detect platform from project files
	if platform == "" {
		detection := detect.Detect(paths.GetRunDirPath())
		if detection.Detected {
			detectedInfo := detection.Platform
			if detection.PlatformVersion != "" {
				detectedInfo += " " + detection.PlatformVersion
			}
			if detection.Language != "" {
				detectedInfo += " (language: " + detection.Language + ")"
			}
			fmtc.SuccessIconLn(fmt.Sprintf("Detected: %s", detectedInfo))
			fmtc.PrintKeyValue("Source", detection.Source)
			fmt.Println("")

			// Auto-confirm with --yes flag
			if args.Yes || fmtc.Confirm("Use detected configuration?", true) {
				platform = detection.Platform
				detectedVersion = detection.PlatformVersion
				detectedLanguage = detection.Language
			}
			fmt.Println("")
		}
	}

	if platform == "" {
		platform = tools.Platform(setupreg.PlatformNames())
	}

	// Determine the language for the project
	language := args.Language
	if language == "" && detectedLanguage != "" {
		language = detectedLanguage
	}
	if info, ok := setupreg.GetPlatformInfo(platform); ok {
		if info.Language != "" {
			language = info.Language
		} else {
			if language == "" && continueSetup {
				language = tools.Language()
			}
			if language == "" {
				language = "php"
			}
		}
	}

	if handler, ok := setupreg.Get(platform); ok {
		handler.Execute(&setupreg.SetupContext{
			ProjectName:     projectName,
			ProjectConf:     projectConf,
			ContinueSetup:  continueSetup,
			Args:            args,
			DetectedVersion: detectedVersion,
			Language:        language,
		})
	} else {
		fmtc.ErrorLn("Unknown platform: " + platform)
	}
}
