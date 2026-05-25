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
	"github.com/faradey/madock/v3/src/model/versions/bigcommerce"
	setupreg "github.com/faradey/madock/v3/src/setup"
)

type Handler struct{}

func (h *Handler) Execute(ctx *setupreg.SetupContext) {
	Execute(ctx.ProjectName, ctx.ProjectConf, ctx.ContinueSetup, ctx.Args)
}

func init() {
	setupreg.Register(setupreg.PlatformInfo{
		Name:        "bigcommerce",
		DisplayName: "BigCommerce",
		Language:    "nodejs",
		Order:       35,
	}, &Handler{})
}

func Execute(projectName string, projectConf map[string]string, continueSetup bool, args *arg_struct.ControllerGeneralSetup) {
	toolsDefVersions := bigcommerce.GetVersions()
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
		presets := preset.GetBigcommercePresets()
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
		fmtc.TitleLn("Choose a BigCommerce SDK preset:")
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
	toolsDefVersions.Platform = "bigcommerce"

	presetCode := toolsDefVersions.PlatformVersion
	if presetCode == "" {
		presetCode = "catalyst"
		toolsDefVersions.PlatformVersion = presetCode
	}
	switch presetCode {
	case "api-php":
		toolsDefVersions.Language = "php"
	default:
		toolsDefVersions.Language = "nodejs"
	}

	fmt.Println("")
	if usePreset {
		fmtc.Title("BigCommerce preset: " + presetCode)
	}

	if !usePreset {
		switch presetCode {
		case "api-php":
			if args.Php == "" {
				tools.Php(&toolsDefVersions.Php)
			} else {
				toolsDefVersions.Php = args.Php
			}
			if args.Db == "" {
				tools.DbMysql(&toolsDefVersions.Db)
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
		default:
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
		DownloadBigcommerce(presetCode)
	}

	if args.Download || args.Install || continueSetup {
		rebuild.Execute()
	}

	if args.Install {
		install.Bigcommerce(projectName, presetCode)
	}
}

// DownloadBigcommerce scaffolds the project based on the selected
// preset. Each preset clones / scaffolds a different upstream
// template — catalyst via npx, stencil via git clone, api-php via
// composer init, app-node via git clone.
func DownloadBigcommerce(presetCode string) {
	target := paths.GetRunDirPath()
	if !isDirEmpty(target) {
		fmtc.WarningLn("Skipping download — project directory is not empty: " + target)
		return
	}

	var cmd *exec.Cmd
	switch presetCode {
	case "catalyst":
		fmtc.InfoIconLn("Cloning BigCommerce Catalyst storefront into " + target)
		// `npm create @bigcommerce/catalyst@latest` parses
		// arguments inconsistently across releases — clone the
		// upstream monorepo's `apps/core` template directly. Easy
		// to maintain, no CLI version pinning required.
		cmd = exec.Command("git", "clone", "--depth", "1",
			"https://github.com/bigcommerce/catalyst.git", ".")
	case "stencil":
		fmtc.InfoIconLn("Cloning BigCommerce Cornerstone theme into " + target)
		// Stencil CLI works against any Cornerstone-based theme.
		// Cornerstone is the canonical starting point.
		cmd = exec.Command("git", "clone", "--depth", "1",
			"https://github.com/bigcommerce/cornerstone.git", ".")
	case "app-node":
		fmtc.InfoIconLn("Cloning BigCommerce sample Node app into " + target)
		// Express + Next.js embedded app — official sample.
		cmd = exec.Command("git", "clone", "--depth", "1",
			"https://github.com/bigcommerce/sample-app-nodejs.git", ".")
	default:
		// api-php
		fmtc.InfoIconLn("Initialising bigcommerce/api-client project in " + target)
		cmd = exec.Command("composer", "init", "--no-interaction",
			"--name=bigcommerce/api-project",
			"--type=project",
			"--require=bigcommerce/api-client:^0.4",
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
	presets := preset.GetBigcommercePresets()

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
		"next":       "catalyst",
		"storefront": "catalyst",
		"theme":      "stencil",
		"php":        "api-php",
		"api":        "api-php",
		"app":        "app-node",
		"node":       "app-node",
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
