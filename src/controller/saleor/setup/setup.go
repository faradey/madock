package setup

import (
	"fmt"
	"strings"

	"github.com/faradey/madock/v3/src/controller/general/rebuild"
	"github.com/faradey/madock/v3/src/helper/cli/arg_struct"
	"github.com/faradey/madock/v3/src/helper/cli/fmtc"
	"github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/configs/projects"
	"github.com/faradey/madock/v3/src/helper/paths"
	"github.com/faradey/madock/v3/src/helper/preset"
	"github.com/faradey/madock/v3/src/helper/setup/tools"
	"github.com/faradey/madock/v3/src/model/versions/saleor"
	setupreg "github.com/faradey/madock/v3/src/setup"
)

type Handler struct{}

func (h *Handler) Execute(ctx *setupreg.SetupContext) {
	Execute(ctx.ProjectName, ctx.ProjectConf, ctx.ContinueSetup, ctx.Args, ctx.DetectedVersion)
}

func init() {
	setupreg.Register(setupreg.PlatformInfo{
		Name:        "saleor",
		DisplayName: "Saleor",
		Language:    "python",
		Order:       45,
	}, &Handler{})
}

func Execute(projectName string, projectConf map[string]string, continueSetup bool, args *arg_struct.ControllerGeneralSetup, detectedVersion string) {
	toolsDefVersions := saleor.GetVersions()
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

	if !usePreset && detectedVersion != "" {
		toolsDefVersions.PlatformVersion = detectedVersion
		fmtc.InfoIconLn(fmt.Sprintf("Using detected Saleor version: %s", detectedVersion))
	}

	if !usePreset && detectedVersion == "" && args.PlatformVersion == "" && continueSetup {
		presets := preset.GetSaleorPresets()
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
		fmtc.TitleLn("Choose a configuration preset:")
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
	toolsDefVersions.Platform = "saleor"
	toolsDefVersions.Language = "python"
	toolsDefVersions.DbType = "PostgreSQL"

	fmt.Println("")
	if usePreset {
		fmtc.Title("Saleor version: " + toolsDefVersions.PlatformVersion)
	}

	if !usePreset {
		if args.Db == "" {
			tools.DbPostgresql(&toolsDefVersions.Db)
		} else {
			toolsDefVersions.Db = args.Db
		}

		if args.Redis == "" {
			tools.Redis(&toolsDefVersions.Redis)
		} else {
			toolsDefVersions.Redis = args.Redis
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

	rebuild.Execute()
}

func findPresetByName(name string) *preset.Preset {
	name = strings.ToLower(name)
	presets := preset.GetSaleorPresets()

	for _, p := range presets {
		if strings.ToLower(p.Name) == name {
			return &p
		}
	}
	for _, p := range presets {
		lowerName := strings.ToLower(p.Name)
		if strings.Contains(lowerName, name) ||
			strings.Contains(p.Versions.PlatformVersion, name) {
			return &p
		}
	}
	aliases := map[string]string{
		"latest": "3.23",
		"stable": "3.20",
		"3":      "3.23",
	}
	if alias, ok := aliases[name]; ok {
		for _, p := range presets {
			if strings.Contains(p.Name, alias) {
				return &p
			}
		}
	}
	return nil
}
