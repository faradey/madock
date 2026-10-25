package setup

import (
	"fmt"
	"strings"

	"github.com/faradey/madock/v4/src/command"
	"github.com/faradey/madock/v4/src/helper/cli/arg_struct"
	"github.com/faradey/madock/v4/src/helper/cli/attr"
	"github.com/faradey/madock/v4/src/helper/cli/fmtc"
	configs2 "github.com/faradey/madock/v4/src/helper/configs"
	"github.com/faradey/madock/v4/src/helper/detect"
	"github.com/faradey/madock/v4/src/helper/logger"
	"github.com/faradey/madock/v4/src/helper/paths"
	"github.com/faradey/madock/v4/src/helper/setup/tools"
	setupreg "github.com/faradey/madock/v4/src/setup"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"setup"},
		Handler:  Execute,
		Help:     "Setup project",
		Category: "setup",
		ArgsType: new(arg_struct.ControllerGeneralSetup),
		// Global: this is the command that turns a directory into a project.
		Global: true,
	})
}

func Execute() {
	args := attr.Parse(new(arg_struct.ControllerGeneralSetup)).(*arg_struct.ControllerGeneralSetup)

	// Propagate -y to per-option selectors so SelectInteractive does not block
	// on stdin for tool versions (Redis/Valkey/Artemis/etc.).
	tools.SetNonInteractive(args.Yes)
	defer tools.SetNonInteractive(false)

	// The engine is the one question --yes could not answer: --db picks a
	// version, so every scripted setup got MariaDB whatever it needed.
	tools.SetDbEngine(args.DbType)
	defer tools.SetDbEngine("")

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

	// Only read here, never made: a setup refused further on — a --preset that
	// matches nothing, an --allow-advisory madock Pro does not take — left an
	// empty projects/<name> behind, an entry for a project that never was.
	// The directory is made when the configuration is saved (SetEnvForProject).
	envFile := paths.GetExecDirPath() + "/projects/" + projectName + "/config.xml"
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
	} else if args.PlatformVersion == "" {
		// --platform named, no version: the version is still the code's. It
		// was not read at all here, so `setup --platform magento2` on a
		// 2.4.8-p5 shop got the newest release's defaults — PHP 8.5, which the
		// shop's lock refuses (found by the Indemorio engine, 2026-10-25). A
		// version given on the command line is left alone: the platform's
		// handler would otherwise prefer the detected one.
		detection := detect.Detect(paths.GetRunDirPath())
		switch {
		case detection.Detected && detection.Platform == platform && detection.PlatformVersion != "":
			detectedVersion = detection.PlatformVersion
			fmtc.SuccessIconLn(fmt.Sprintf("Detected %s %s from %s", platform, detection.PlatformVersion, detection.Source))
		case detection.Detected && detection.Platform != platform && detection.Platform != "custom":
			fmtc.WarningLn(fmt.Sprintf("The code here looks like %s (%s), not %s; setting up %s as asked.", detection.Platform, detection.Source, platform, platform))
		}
	}

	if platform == "" {
		infos := setupreg.Platforms()
		choices := make([]tools.PlatformChoice, len(infos))
		for i, p := range infos {
			choices[i] = tools.PlatformChoice{Name: p.Name, DisplayName: p.DisplayName}
		}
		platform = tools.Platform(choices)
	}

	if hasConfig && continueSetup {
		tools.SetReconfigure(true)
		defer tools.SetReconfigure(false)
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
			configLanguage := projectConf["language"]
			if language == "" && continueSetup {
				language = tools.Language(configLanguage)
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
			ContinueSetup:   continueSetup,
			Args:            args,
			DetectedVersion: detectedVersion,
			Language:        language,
		})
	} else {
		fmtc.ErrorLn("Unknown platform: " + platform)
	}
}
