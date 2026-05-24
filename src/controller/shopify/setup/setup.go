package setup

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/faradey/madock/v3/src/controller/general/install"
	"github.com/faradey/madock/v3/src/controller/general/rebuild"
	"github.com/faradey/madock/v3/src/helper/cli/arg_struct"
	"github.com/faradey/madock/v3/src/helper/cli/fmtc"
	"github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/configs/projects"
	"github.com/faradey/madock/v3/src/helper/paths"
	"github.com/faradey/madock/v3/src/helper/preset"
	"github.com/faradey/madock/v3/src/helper/setup/tools"
	"github.com/faradey/madock/v3/src/model/versions/shopify"
	setupreg "github.com/faradey/madock/v3/src/setup"
)

type Handler struct{}

func (h *Handler) Execute(ctx *setupreg.SetupContext) {
	Execute(ctx.ProjectName, ctx.ProjectConf, ctx.ContinueSetup, ctx.Args)
}

func init() {
	setupreg.Register(setupreg.PlatformInfo{
		Name:        "shopify",
		DisplayName: "Shopify",
		Language:    "php",
		Order:       30,
	}, &Handler{})
}

func Execute(projectName string, projectConf map[string]string, continueSetup bool, args *arg_struct.ControllerGeneralSetup) {
	toolsDefVersions := shopify.GetVersions()
	usePreset := false

	if args.Preset != "" {
		if selected := findPresetByName(args.Preset); selected != nil {
			toolsDefVersions = selected.Versions
			usePreset = true
			fmtc.SuccessIconLn(fmt.Sprintf("Using preset: %s", selected.Name))
		} else {
			fmtc.WarningLn(fmt.Sprintf("Preset '%s' not found, proceeding with manual configuration", args.Preset))
		}
	}

	if !usePreset && args.PlatformVersion == "" && continueSetup {
		presets := preset.GetShopifyPresets()
		presetOptions := make([]fmtc.PresetOption, 0, len(presets)+1)
		for _, p := range presets {
			presetOptions = append(presetOptions, fmtc.PresetOption{
				Name:        p.Name,
				Description: p.Description,
				IsCustom:    false,
			})
		}
		presetOptions = append(presetOptions, fmtc.PresetOption{
			Name:        preset.CustomPreset.Name,
			Description: preset.CustomPreset.Description,
			IsCustom:    true,
		})

		fmt.Println("")
		fmtc.TitleLn("Choose a Shopify SDK preset:")
		selectedIdx := fmtc.SelectPreset("Configuration", presetOptions)
		if selectedIdx < len(presets) {
			toolsDefVersions = presets[selectedIdx].Versions
			usePreset = true
			fmt.Println("")
			fmtc.SuccessIconLn(fmt.Sprintf("Using preset: %s", presets[selectedIdx].Name))
		}
	}

	if !continueSetup {
		return
	}

	if !usePreset {
		tools.PopulateFromConfig(&toolsDefVersions, projectConf)
	}
	toolsDefVersions.Platform = "shopify"

	presetCode := toolsDefVersions.PlatformVersion
	if presetCode == "" {
		presetCode = "api-php"
		toolsDefVersions.PlatformVersion = presetCode
	}
	switch presetCode {
	case "hydrogen", "app-remix":
		toolsDefVersions.Language = "nodejs"
	default:
		toolsDefVersions.Language = "php"
	}

	fmt.Println("")
	if usePreset {
		fmtc.Title("Shopify preset: " + presetCode)
	}

	if !usePreset {
		// Manual mode — keep the legacy interactive flow but skip
		// PHP/DB prompts for Node-only presets.
		switch presetCode {
		case "hydrogen", "app-remix":
			if args.NodeJs == "" {
				tools.NodeJs(&toolsDefVersions.NodeJs)
			} else {
				toolsDefVersions.NodeJs = args.NodeJs
			}
			if args.Yarn == "" {
				tools.Yarn(&toolsDefVersions.Yarn)
			} else {
				toolsDefVersions.Yarn = args.Yarn
			}
		default:
			if args.Php == "" {
				tools.Php(&toolsDefVersions.Php)
			} else {
				toolsDefVersions.Php = args.Php
			}
			if args.Db == "" {
				tools.DbEngine(&toolsDefVersions.DbType)
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
			} else {
				toolsDefVersions.Db = args.Db
			}
			if args.Composer == "" {
				tools.Composer(&toolsDefVersions.Composer)
			} else {
				toolsDefVersions.Composer = args.Composer
			}
			if args.Redis == "" {
				tools.Redis(&toolsDefVersions.Redis)
			} else {
				toolsDefVersions.Redis = args.Redis
			}
			if presetCode == "laravel-shopify" {
				if args.NodeJs == "" {
					tools.NodeJs(&toolsDefVersions.NodeJs)
				} else {
					toolsDefVersions.NodeJs = args.NodeJs
				}
				if args.Yarn == "" {
					tools.Yarn(&toolsDefVersions.Yarn)
				} else {
					toolsDefVersions.Yarn = args.Yarn
				}
			}
		}
	}

	if args.Hosts == "" {
		tools.Hosts(projectName, &toolsDefVersions.Hosts, projectConf)
	} else {
		toolsDefVersions.Hosts = args.Hosts
	}

	projects.SetEnvForProject(projectName, toolsDefVersions, configs.GetProjectConfigOnly(projectName))
	paths.MakeDirsByPath(paths.GetExecDirPath() + "/projects/" + projectName + "/backup/db")

	fmtc.SuccessLn("\n" + "Finish set up environment")
	fmtc.ToDoLn("Optionally, you can configure SSH access to the development server in order ")
	fmtc.ToDoLn("to synchronize the database and media files. Enter SSH data in ")
	fmtc.ToDoLn(paths.GetExecDirPath() + "/projects/" + projectName + "/config.xml")

	if args.Download {
		DownloadShopify(presetCode)
	}

	if args.Download || args.Install || continueSetup {
		rebuild.Execute()
	}

	if args.Install {
		install.Shopify(projectName, presetCode)
	}
}

// DownloadShopify scaffolds the project based on the selected preset.
// Hydrogen / app-remix run their official npx scaffolders; api-php and
// laravel-shopify use composer create-project / init.
func DownloadShopify(presetCode string) {
	target := paths.GetRunDirPath()
	if !isDirEmpty(target) {
		fmtc.WarningLn("Skipping download — project directory is not empty: " + target)
		return
	}

	var cmd *exec.Cmd
	switch presetCode {
	case "hydrogen":
		fmtc.InfoIconLn("Scaffolding Hydrogen storefront into " + target)
		// `npm create @shopify/hydrogen@latest` runs the CLI in
		// quickstart mode; the storefront ends up in ./<name>/ so we
		// move files up after the clone.
		cmd = exec.Command("npm", "create", "-y", "@shopify/hydrogen@latest", "--",
			"--path", ".", "--quickstart", "--language", "ts", "--no-install-deps")
	case "app-remix":
		fmtc.InfoIconLn("Scaffolding Shopify App (Remix) into " + target)
		cmd = exec.Command("npm", "init", "-y", "@shopify/app@latest", "--",
			"--name", "shopify-app", "--template", "remix", "--no-install-deps")
	case "laravel-shopify":
		fmtc.InfoIconLn("Scaffolding Laravel + Kyon147/laravel-shopify into " + target)
		// Two-step: laravel skeleton, then composer require the
		// shopify package. Both run inside the host since the PHP
		// container isn't up yet.
		cmd = exec.Command("composer", "create-project", "--no-install",
			"laravel/laravel", ".")
	default:
		fmtc.InfoIconLn("Initialising shopify-api-php project in " + target)
		cmd = exec.Command("composer", "init", "--no-interaction",
			"--name=shopify/api-project",
			"--type=project",
			"--require=shopify/shopify-api:^7.0",
			"--stability=stable")
	}
	cmd.Dir = target
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmtc.WarningLn("Download failed: " + err.Error() + ". You can scaffold the project manually inside " + target + " and re-run `madock install`.")
	}
}

func isDirEmpty(path string) bool {
	entries, err := os.ReadDir(path)
	if err != nil {
		return true
	}
	for _, e := range entries {
		if e.Name() == ".madock" || e.Name() == "." || e.Name() == ".." {
			continue
		}
		return false
	}
	return true
}

func findPresetByName(name string) *preset.Preset {
	name = strings.ToLower(name)
	presets := preset.GetShopifyPresets()

	for _, p := range presets {
		if strings.ToLower(p.Name) == name ||
			strings.ToLower(p.Versions.PlatformVersion) == name {
			return &p
		}
	}
	for _, p := range presets {
		lowerName := strings.ToLower(p.Name)
		lowerVer := strings.ToLower(p.Versions.PlatformVersion)
		if strings.Contains(lowerName, name) || strings.Contains(lowerVer, name) {
			return &p
		}
	}
	aliases := map[string]string{
		"node":       "hydrogen",
		"storefront": "hydrogen",
		"app":        "app-remix",
		"remix":      "app-remix",
		"php":        "api-php",
		"api":        "api-php",
		"laravel":    "laravel-shopify",
	}
	if alias, ok := aliases[name]; ok {
		for _, p := range presets {
			if p.Versions.PlatformVersion == alias {
				return &p
			}
		}
	}
	return nil
}
