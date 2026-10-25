package setup

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/faradey/madock/v4/src/controller/general/install"
	"github.com/faradey/madock/v4/src/controller/general/rebuild"
	"github.com/faradey/madock/v4/src/helper/cli"
	"github.com/faradey/madock/v4/src/helper/cli/arg_struct"
	"github.com/faradey/madock/v4/src/helper/cli/fmtc"
	"github.com/faradey/madock/v4/src/helper/composerpolicy"
	configs2 "github.com/faradey/madock/v4/src/helper/configs"
	"github.com/faradey/madock/v4/src/helper/configs/projects"
	"github.com/faradey/madock/v4/src/helper/docker"
	"github.com/faradey/madock/v4/src/helper/logger"
	"github.com/faradey/madock/v4/src/helper/paths"
	"github.com/faradey/madock/v4/src/helper/preset"
	"github.com/faradey/madock/v4/src/helper/setup/tools"
	"github.com/faradey/madock/v4/src/model/versions"
	"github.com/faradey/madock/v4/src/model/versions/magento2"
	setupreg "github.com/faradey/madock/v4/src/setup"
)

type Handler struct{}

func (h *Handler) Execute(ctx *setupreg.SetupContext) {
	ExecuteWithVersion(ctx.ProjectName, ctx.ProjectConf, ctx.ContinueSetup, ctx.Args, ctx.DetectedVersion)
}

func init() {
	setupreg.Register(setupreg.PlatformInfo{
		Name:        "magento2",
		DisplayName: "Magento 2",
		Language:    "php",
		Order:       20,
	}, &Handler{})
}

func ExecuteWithVersion(projectName string, projectConf map[string]string, continueSetup bool, args *arg_struct.ControllerGeneralSetup, detectedVersion string) {
	toolsDefVersions := magento2.GetVersions("")

	// Before anything is written: an id that is no id, or an edition that
	// does not accept advisories at all, refuses the whole setup.
	advisories, err := composerpolicy.Parse(args.AllowAdvisory)
	if err != nil {
		fmtc.ErrorLn(err.Error())
		os.Exit(1)
	}
	if len(advisories) > 0 && composerpolicy.Refusal != nil {
		if why := composerpolicy.Refusal(); why != "" {
			fmtc.ErrorLn(why)
			os.Exit(1)
		}
	}

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
			fmtc.ErrorLn(preset.Unmatched(args.Preset, "magento2"))
			os.Exit(1)
		}
	}

	// Use detected version if available
	if !usePreset && detectedVersion != "" {
		mageVersion = detectedVersion
		toolsDefVersions = magento2.GetVersions(mageVersion)
		fmtc.InfoIconLn(fmt.Sprintf("Using detected Magento version: %s", mageVersion))
	} else if !usePreset && args.PlatformVersion != "" {
		mageVersion = args.PlatformVersion
		// The same table the detected-version branch above reads. Naming the
		// version on the command line used to skip it entirely and keep the
		// defaults for an unknown version — where Composer is the empty string,
		// because GetComposerVersion("") matches none of its ranges. The result
		// was a 2.4.8 installed with Composer 1, which cannot resolve
		// magento/product-community-edition at all: every package "could not be
		// found", after a twenty-minute download.
		//
		// Explicit flags still win: each is applied over this a few steps below.
		toolsDefVersions = magento2.GetVersions(mageVersion)
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
		"RabbitMQ Version",
		"Hosts Configuration",
	}
	tools.InitProgress(setupSteps)
	currentStep := 1

	if !usePreset && toolsDefVersions.Php == "" && detectedVersion == "" {
		if mageVersion == "" {
			tools.SetProgressStep(currentStep)
			mageVersion = tools.RequireAnswer("Specify Magento version", "--platform-version")
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
		if !usePreset {
			tools.PopulateFromConfig(&toolsDefVersions, projectConf)
		}

		fmt.Println("")
		fmtc.Title("Your Magento version is " + toolsDefVersions.PlatformVersion)

		if usePreset {
			// Skip to hosts configuration when using preset
			currentStep = 9
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

			// Step 3: Database Engine & Version
			tools.SetProgressStep(currentStep)
			if args.Db == "" || tools.HasDbEngineOverride() {
				tools.DbEngine(&toolsDefVersions.DbType)
				// An explicit version is taken as given; the selectors are
				// only for deciding one.
				if args.Db != "" {
					toolsDefVersions.Db = args.Db
				} else {
					switch toolsDefVersions.DbType {
					case "MySQL":
						tools.DbMysql(&toolsDefVersions.Db)
					case "PostgreSQL":
						tools.DbPostgresql(&toolsDefVersions.Db)
					case "MongoDB":
						tools.DbMongodb(&toolsDefVersions.Db)
					default:
						tools.Db(&toolsDefVersions.Db)
					}
				}
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
			if strings.EqualFold(toolsDefVersions.SearchEngine, "Elasticsearch") {
				if args.SearchEngineVersion == "" {
					tools.Elastic(&toolsDefVersions.Elastic)
				} else {
					toolsDefVersions.Elastic = args.SearchEngineVersion
				}
			} else if strings.EqualFold(toolsDefVersions.SearchEngine, "OpenSearch") {
				if refusal := openSearchRefusal(toolsDefVersions, args.SearchEngineVersion); refusal != "" {
					fmtc.ErrorLn(refusal)
					os.Exit(1)
				}
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

			// Step 8: RabbitMQ Version
			tools.SetProgressStep(currentStep)
			if args.RabbitMQ == "" {
				tools.RabbitMQ(&toolsDefVersions.RabbitMQ)
			} else {
				toolsDefVersions.RabbitMQ = args.RabbitMQ
			}
			currentStep++

			// Step 9: Hosts Configuration
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

		// Ask for confirmation (skip with --yes flag)
		fmt.Println("")
		if !args.Yes && !fmtc.Confirm("Proceed with this configuration?", true) {
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

		if args.Download && args.PlatformEdition == "" && !args.Yes {
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
		// When --yes is used, default to community edition
	}

	if args.Download || args.Install || continueSetup {
		rebuild.Execute()
	}

	if args.Download {
		if why := composerBlocks(mageVersion); why != "" && len(advisories) == 0 {
			fmtc.WarningLn(why)
		}
		DownloadMagento(projectName, edition, mageVersion, args.SampleData, advisories)
	}

	if args.Install {
		install.Magento(projectName, toolsDefVersions.PlatformVersion)
	}
}

func DownloadMagento(projectName, edition, version string, isSampleData bool, advisories []string) {
	projectConf := configs2.GetCurrentProjectConfig()
	service, user, workdir := cli.GetEnvForUserServiceWorkdir("php", "www-data", projectConf["workdir"])
	ttyFlag := "-i"
	if docker.IsTTYAvailable() {
		ttyFlag = "-it"
	}
	command := []string{
		"exec",
		ttyFlag,
		"-u",
		user,
		docker.GetContainerName(projectConf, projectName, service),
		"bash",
		"-c",
		downloadScript(workdir, edition, version, isSampleData, advisories),
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
					{Key: "Database", Value: dbDisplayName(v)},
					{Key: "Composer", Value: v.Composer},
				},
			},
			{
				Name:  "Search Engine",
				Items: getSearchEngineItems(v),
			},
			{
				Name: "Cache & Queue",
				Items: []fmtc.SectionItem{
					{Key: "Redis", Value: v.Redis},
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
	if strings.EqualFold(v.SearchEngine, "Elasticsearch") {
		return []fmtc.SectionItem{
			{Key: "Engine", Value: "Elasticsearch"},
			{Key: "Version", Value: v.Elastic},
		}
	} else if strings.EqualFold(v.SearchEngine, "OpenSearch") {
		return []fmtc.SectionItem{
			{Key: "Engine", Value: "OpenSearch"},
			{Key: "Version", Value: v.OpenSearch},
		}
	}
	return []fmtc.SectionItem{
		{Key: "Engine", Value: "Not configured"},
	}
}

func dbDisplayName(v versions.ToolsVersions) string {
	switch v.DbType {
	case "MySQL":
		return "MySQL " + v.Db
	case "PostgreSQL":
		return "PostgreSQL " + v.Db
	case "MongoDB":
		return "MongoDB " + v.Db
	default:
		return "MariaDB " + v.Db
	}
}

// downloadScript is the shell line that downloads Magento into workdir.
//
// With advisories, create-project only writes the project's files
// (--no-install), the ids go into its composer.json as audit.ignore, and the
// install that follows reads them: create-project resolves before any
// composer.json of the project exists, so there is nowhere earlier to put
// them. Without, the line is what it always was.
func downloadScript(workdir, edition, version string, sampleData bool, advisories []string) string {
	tmp := workdir + "/download-magento123456789"
	createProject := "composer create-project --repository-url=https://repo.magento.com/ magento/project-" + edition + "-edition:" + version + " ./download-magento123456789 "
	if cfg := composerpolicy.ConfigCommand(advisories); cfg != "" {
		createProject = "composer create-project --no-install --repository-url=https://repo.magento.com/ magento/project-" + edition + "-edition:" + version + " ./download-magento123456789 " +
			"&& (cd ./download-magento123456789 && " + cfg + ") "
	}
	script := "cd " + workdir + " " +
		"&& rm -r -f " + tmp + " " +
		"&& mkdir " + tmp + " " +
		"&& " + createProject +
		"&& shopt -s dotglob " +
		"&& mv  -v ./download-magento123456789/* ./ " +
		"&& rm -r -f ./download-magento123456789 " +
		"&& composer install"
	if sampleData {
		script += " && bin/magento sampledata:deploy"
	}
	return script
}

// composerBlocks says why Composer will refuse to install this Magento
// release, or nothing.
//
// Measured on 2026-10-25 against packagist's advisories and repo.magento.com's
// metadata: every product-community-edition below 2.4.8 requires
// league/flysystem 1.x or 2.x, and PKSA-w9tt-7782-78jx (published 2026-09-29)
// covers every flysystem up to 3.35.2 — 1.x and 2.x have no fixed release, the
// fix is 3.36.0, which only 2.4.8 and later accept (^3.0). Composer 2.8+
// blocks a package with an advisory, so a download or composer update of an
// older release fails. Not bypassed: that would mean installing the
// vulnerable package on purpose, which is the owner's decision, not setup's.
// An existing project of such a release still starts and runs.
func composerBlocks(version string) string {
	if version == "" || !versions.AtLeast(version, "2.0.0") || versions.AtLeast(version, "2.4.8") {
		return ""
	}
	return "Composer refuses to install Magento " + version + ": every release below 2.4.8 needs league/flysystem 1.x or 2.x, " +
		"which advisory PKSA-w9tt-7782-78jx covers with no fixed 1.x or 2.x release (the fix, 3.36.0, needs Magento 2.4.8+). " +
		"The download below will fail; choose 2.4.8 or later, or accept those advisories for this project on purpose: " +
		"--allow-advisory PKSA-w9tt-7782-78jx,PKSA-db8d-773v-rd1n."
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

// openSearchRefusal says why OpenSearch cannot be the search engine of this
// Magento release, or nothing when it can.
//
// The matrix answers "NotCompatible" for a release older than 2.4.3-p2, and
// that answer was never read: it became the pre-selected version, and under
// -y the version, written into the config as the OpenSearch image tag. The
// project then failed at the image pull. A version given explicitly is still
// taken at its word — whoever names one knows something the matrix does not.
func openSearchRefusal(tv versions.ToolsVersions, explicitVersion string) string {
	if explicitVersion != "" || tv.OpenSearch != versions.NotCompatible {
		return ""
	}
	return "OpenSearch does not support Magento " + tv.PlatformVersion +
		" — it is supported from 2.4.3-p2 (2.3.7-p3 and 2.3.7-p4 on the 2.3 line). Choose Elasticsearch, or name a version with --search-engine-version."
}
