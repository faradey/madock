package setup

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/faradey/madock/src/controller/general/install"
	"github.com/faradey/madock/src/controller/general/rebuild"
	"github.com/faradey/madock/src/helper/cli"
	"github.com/faradey/madock/src/helper/cli/arg_struct"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	configs2 "github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/configs/projects"
	"github.com/faradey/madock/src/helper/docker"
	"github.com/faradey/madock/src/helper/logger"
	"github.com/faradey/madock/src/helper/paths"
	"github.com/faradey/madock/src/helper/preset"
	"github.com/faradey/madock/src/helper/setup/tools"
	"github.com/faradey/madock/src/model/versions"
	"github.com/faradey/madock/src/model/versions/magento2"
)

func ExecuteWithVersion(projectName string, projectConf map[string]string, continueSetup bool, args *arg_struct.ControllerGeneralSetup, detectedVersion string) {
	toolsDefVersions := magento2.GetVersions("")

	mageVersion := ""
	usePreset := false

	// Check if preset is specified via command line
	if args.Preset != "" {
		selectedPreset := findPresetByName(args.Preset)
		if selectedPreset != nil {
			toolsDefVersions = selectedPreset.Versions
			mageVersion = toolsDefVersions.PlatformVersion
			usePreset = true
			fmtc.SuccessIconLn(fmt.Sprintf("Using preset: %s", selectedPreset.Name))
		} else {
			fmtc.WarningLn(fmt.Sprintf("Preset '%s' not found, proceeding with manual configuration", args.Preset))
		}
	}

	// Use detected version if available
	if !usePreset && detectedVersion != "" {
		mageVersion = detectedVersion
		toolsDefVersions = magento2.GetVersions(mageVersion)
		fmtc.InfoIconLn(fmt.Sprintf("Using detected Magento version: %s", mageVersion))
	} else if !usePreset && args.PlatformVersion != "" {
		mageVersion = args.PlatformVersion
		if args.Php != "" {
			toolsDefVersions.Php = args.Php
		}
	}

	// If no preset and no detected version, offer preset selection
	if !usePreset && detectedVersion == "" && args.PlatformVersion == "" && continueSetup {
		presets := preset.GetMagentoPresets()
		presetOptions := make([]fmtc.PresetOption, 0, len(presets)+1)

		for _, p := range presets {
			presetOptions = append(presetOptions, fmtc.PresetOption{
				Name:        p.Name,
				Description: p.Description,
				IsCustom:    false,
			})
		}

		// Add custom option at the end
		presetOptions = append(presetOptions, fmtc.PresetOption{
			Name:        preset.CustomPreset.Name,
			Description: preset.CustomPreset.Description,
			IsCustom:    true,
		})

		fmt.Println("")
		fmtc.TitleLn("Choose a configuration preset:")
		selectedIdx := fmtc.SelectPreset("Configuration", presetOptions)

		if selectedIdx < len(presets) {
			// User selected a preset
			selectedPreset := presets[selectedIdx]
			toolsDefVersions = selectedPreset.Versions
			mageVersion = toolsDefVersions.PlatformVersion
			usePreset = true
			fmt.Println("")
			fmtc.SuccessIconLn(fmt.Sprintf("Using preset: %s", selectedPreset.Name))
		}
		// If selectedIdx == len(presets), user selected "Custom", continue with manual selection
	}

	// Initialize progress tracker for setup steps
	setupSteps := []string{
		"Magento Version",
		"PHP Version",
		"Database Version",
		"Composer Version",
		"Search Engine",
		"Search Engine Version",
		"Redis Version",
		"Valkey Version",
		"RabbitMQ Version",
		"Hosts Configuration",
	}
	tools.InitProgress(setupSteps)
	currentStep := 1

	if !usePreset && toolsDefVersions.Php == "" && detectedVersion == "" {
		if mageVersion == "" {
			tools.SetProgressStep(currentStep)
			fmtc.Title("Specify Magento version: ")
			mageVersion, _ = tools.Waiter()
		}
		if mageVersion != "" {
			toolsDefVersions = magento2.GetVersions(mageVersion)
		} else {
			ExecuteWithVersion(projectName, projectConf, continueSetup, args, "")
			return
		}
	}
	currentStep++

	edition := "community"
	if args.PlatformEdition != "" {
		edition = args.PlatformEdition
	}

	if continueSetup {
		fmt.Println("")
		fmtc.Title("Your Magento version is " + toolsDefVersions.PlatformVersion)

		if usePreset {
			// Skip to hosts configuration when using preset
			currentStep = 10
			tools.SetProgressStep(currentStep)
			if args.Hosts == "" {
				tools.Hosts(projectName, &toolsDefVersions.Hosts, projectConf)
			} else {
				toolsDefVersions.Hosts = args.Hosts
			}
		} else {
			// Step 2: PHP Version
			tools.SetProgressStep(currentStep)
			if args.Php == "" {
				tools.Php(&toolsDefVersions.Php)
			} else {
				toolsDefVersions.Php = args.Php
			}
			currentStep++

			// Step 3: Database Version
			tools.SetProgressStep(currentStep)
			if args.Db == "" {
				tools.Db(&toolsDefVersions.Db)
			} else {
				toolsDefVersions.Db = args.Db
			}
			currentStep++

			// Step 4: Composer Version
			tools.SetProgressStep(currentStep)
			if args.Composer == "" {
				tools.Composer(&toolsDefVersions.Composer)
			} else {
				toolsDefVersions.Composer = args.Composer
			}
			currentStep++

			// Step 5: Search Engine
			tools.SetProgressStep(currentStep)
			if args.SearchEngine == "" {
				tools.SearchEngine(&toolsDefVersions.SearchEngine)
			} else {
				toolsDefVersions.SearchEngine = args.SearchEngine
			}
			currentStep++

			// Step 6: Search Engine Version
			tools.SetProgressStep(currentStep)
			if toolsDefVersions.SearchEngine == "Elasticsearch" {
				if args.SearchEngineVersion == "" {
					tools.Elastic(&toolsDefVersions.Elastic)
				} else {
					toolsDefVersions.Elastic = args.SearchEngineVersion
				}
			} else if toolsDefVersions.SearchEngine == "OpenSearch" {
				if args.SearchEngineVersion == "" {
					tools.OpenSearch(&toolsDefVersions.OpenSearch)
				} else {
					toolsDefVersions.OpenSearch = args.SearchEngineVersion
				}
			}
			currentStep++

			// Step 7: Redis Version
			tools.SetProgressStep(currentStep)
			if args.Redis == "" {
				tools.Redis(&toolsDefVersions.Redis)
			} else {
				toolsDefVersions.Redis = args.Redis
			}
			currentStep++

			// Step 8: Valkey Version
			tools.SetProgressStep(currentStep)
			if args.Valkey == "" {
				tools.Valkey(&toolsDefVersions.Valkey)
			} else {
				toolsDefVersions.Valkey = args.Valkey
			}
			currentStep++

			// Step 9: RabbitMQ Version
			tools.SetProgressStep(currentStep)
			if args.RabbitMQ == "" {
				tools.RabbitMQ(&toolsDefVersions.RabbitMQ)
			} else {
				toolsDefVersions.RabbitMQ = args.RabbitMQ
			}
			currentStep++

			// Step 10: Hosts Configuration
			tools.SetProgressStep(currentStep)
			if args.Hosts == "" {
				tools.Hosts(projectName, &toolsDefVersions.Hosts, projectConf)
			} else {
				toolsDefVersions.Hosts = args.Hosts
			}
		}

		// Show completion
		tools.CompleteProgress()

		// Display configuration summary
		displayConfigSummary(toolsDefVersions, projectName)

		// Ask for confirmation
		fmt.Println("")
		if !fmtc.Confirm("Proceed with this configuration?", true) {
			fmtc.WarningLn("Setup cancelled.")
			return
		}

		// Save configuration
		projects.SetEnvForProject(projectName, toolsDefVersions, configs2.GetProjectConfigOnly(projectName))
		paths.MakeDirsByPath(paths.GetExecDirPath() + "/projects/" + projectName + "/backup/db")

		fmt.Println("")
		fmtc.SuccessIconLn("Configuration saved!")
		fmt.Println("")
		fmtc.ToDoLn("Optionally, you can configure SSH access to the development server in order ")
		fmtc.ToDoLn("to synchronize the database and media files. Enter SSH data in ")
		fmtc.ToDoLn(paths.GetExecDirPath() + "/projects/" + projectName + "/config.xml")

		if args.Download && args.PlatformEdition == "" {
			fmt.Println("")
			editions := []string{"Community", "Enterprise"}
			selector := fmtc.NewInteractiveSelector("Magento Edition", editions, 0)
			idx, _ := selector.Run()
			if idx == 0 {
				edition = "community"
			} else {
				edition = "enterprise"
			}
		}
	}

	if args.Download || args.Install || continueSetup {
		rebuild.Execute()
	}

	if args.Download {
		DownloadMagento(projectName, edition, mageVersion, args.SampleData)
	}

	if args.Install {
		install.Magento(projectName, toolsDefVersions.PlatformVersion)
	}
}

func DownloadMagento(projectName, edition, version string, isSampleData bool) {
	projectConf := configs2.GetCurrentProjectConfig()
	sampleData := ""
	if isSampleData {
		sampleData = " && bin/magento sampledata:deploy"
	}
	service, user, workdir := cli.GetEnvForUserServiceWorkdir("php", "www-data", projectConf["workdir"])
	command := []string{
		"exec",
		"-it",
		"-u",
		user,
		docker.GetContainerName(projectConf, projectName, service),
		"bash",
		"-c",
		"cd " + workdir + " " +
			"&& rm -r -f " + workdir + "/download-magento123456789 " +
			"&& mkdir " + workdir + "/download-magento123456789 " +
			"&& composer create-project --repository-url=https://repo.magento.com/ magento/project-" + edition + "-edition:" + version + " ./download-magento123456789 " +
			"&& shopt -s dotglob " +
			"&& mv  -v ./download-magento123456789/* ./ " +
			"&& rm -r -f ./download-magento123456789 " +
			"&& composer install" + sampleData,
	}
	cmd := exec.Command("docker", command...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		logger.Fatal(err)
	}
}

func displayConfigSummary(v versions.ToolsVersions, projectName string) {
	summary := &fmtc.ConfigSummary{
		Title: "Configuration Summary",
		Sections: []fmtc.ConfigSection{
			{
				Name: "Core Services",
				Items: []fmtc.SectionItem{
					{Key: "Platform", Value: "Magento " + v.PlatformVersion},
					{Key: "PHP", Value: v.Php},
					{Key: "Database", Value: "MariaDB " + v.Db},
					{Key: "Composer", Value: v.Composer},
				},
			},
			{
				Name: "Search Engine",
				Items: getSearchEngineItems(v),
			},
			{
				Name: "Cache & Queue",
				Items: []fmtc.SectionItem{
					{Key: "Redis", Value: v.Redis},
					{Key: "Valkey", Value: v.Valkey},
					{Key: "RabbitMQ", Value: v.RabbitMQ},
				},
			},
			{
				Name: "Hosts",
				Items: []fmtc.SectionItem{
					{Key: "Domain", Value: v.Hosts},
				},
			},
		},
	}
	summary.Display()
}

func getSearchEngineItems(v versions.ToolsVersions) []fmtc.SectionItem {
	if v.SearchEngine == "Elasticsearch" {
		return []fmtc.SectionItem{
			{Key: "Engine", Value: "Elasticsearch"},
			{Key: "Version", Value: v.Elastic},
		}
	} else if v.SearchEngine == "OpenSearch" {
		return []fmtc.SectionItem{
			{Key: "Engine", Value: "OpenSearch"},
			{Key: "Version", Value: v.OpenSearch},
		}
	}
	return []fmtc.SectionItem{
		{Key: "Engine", Value: "Not configured"},
	}
}

// findPresetByName finds a preset by name or keyword
func findPresetByName(name string) *preset.Preset {
	name = strings.ToLower(name)
	presets := preset.GetMagentoPresets()

	// Try exact match first
	for _, p := range presets {
		if strings.ToLower(p.Name) == name {
			return &p
		}
	}

	// Try keyword match (e.g., "247" matches "Magento 2.4.7")
	for _, p := range presets {
		lowerName := strings.ToLower(p.Name)
		if strings.Contains(lowerName, name) ||
			strings.Contains(strings.ReplaceAll(p.Versions.PlatformVersion, ".", ""), name) ||
			strings.Contains(p.Versions.PlatformVersion, name) {
			return &p
		}
	}

	// Try common aliases
	aliases := map[string]string{
		"latest":  "2.4.7",
		"lts":     "2.4.6",
		"legacy":  "2.4.5",
		"minimal": "Development",
		"dev":     "Development",
	}

	if alias, ok := aliases[name]; ok {
		for _, p := range presets {
			if strings.Contains(p.Name, alias) || strings.Contains(p.Versions.PlatformVersion, alias) {
				return &p
			}
		}
	}

	return nil
}
