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
	"github.com/faradey/madock/v3/src/model/versions/medusa"
	setupreg "github.com/faradey/madock/v3/src/setup"
)

type Handler struct{}

func (h *Handler) Execute(ctx *setupreg.SetupContext) {
	Execute(ctx.ProjectName, ctx.ProjectConf, ctx.ContinueSetup, ctx.Args, ctx.DetectedVersion)
}

func init() {
	setupreg.Register(setupreg.PlatformInfo{
		Name:        "medusa",
		DisplayName: "Medusa.js",
		Language:    "nodejs",
		Order:       40,
	}, &Handler{})
}

func Execute(projectName string, projectConf map[string]string, continueSetup bool, args *arg_struct.ControllerGeneralSetup, detectedVersion string) {
	toolsDefVersions := medusa.GetVersions()
	usePreset := false

	// CLI --preset
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
		fmtc.InfoIconLn(fmt.Sprintf("Using detected Medusa version: %s", detectedVersion))
	}

	// Interactive preset wizard when no preset / no detection / no version
	if !usePreset && detectedVersion == "" && args.PlatformVersion == "" && continueSetup {
		presets := preset.GetMedusaPresets()
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
	toolsDefVersions.Platform = "medusa"
	toolsDefVersions.Language = "nodejs"
	toolsDefVersions.DbType = "PostgreSQL"

	fmt.Println("")
	if usePreset {
		fmtc.Title("Medusa version: " + toolsDefVersions.PlatformVersion)
	}

	// Preset skips per-service prompts; only ask for hosts.
	if !usePreset {
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

		if args.RabbitMQ == "" {
			tools.RabbitMQ(&toolsDefVersions.RabbitMQ)
		} else {
			toolsDefVersions.RabbitMQ = args.RabbitMQ
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

	if args.Download || args.Install || continueSetup {
		rebuild.Execute()
	}

	if args.Download {
		DownloadMedusa(projectName)
	}

	if args.Install {
		install.Medusa(projectName, toolsDefVersions.PlatformVersion)
	}
}

// DownloadMedusa clones the official Medusa starter into the project
// root when the directory is empty. The user can then run `madock
// install` (or use `-i` on setup) to apply migrations and create the
// admin user.
func DownloadMedusa(projectName string) {
	target := paths.GetRunDirPath()
	if !isDirEmpty(target) {
		fmtc.WarningLn("Skipping download — project directory is not empty: " + target)
		return
	}
	repo := "https://github.com/medusajs/medusa-starter-default.git"
	fmtc.InfoIconLn("Cloning " + repo + " into " + target)
	cmd := exec.Command("git", "clone", "--depth", "1", repo, ".")
	cmd.Dir = target
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmtc.WarningLn("Failed to clone Medusa starter: " + err.Error())
	}
}

// isDirEmpty returns true when path doesn't exist or holds no entries
// besides dotfiles madock itself may have created (.madock/).
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

// findPresetByName resolves --preset value to a Medusa preset.
// Accepts exact names, fragments, and aliases like "latest", "stable", "legacy", "v2", "v1".
func findPresetByName(name string) *preset.Preset {
	name = strings.ToLower(name)
	presets := preset.GetMedusaPresets()

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
		"latest": "2.x",
		"stable": "2.0",
		"legacy": "1.x",
		"v2":     "2.x",
		"v1":     "1.x",
		"2":      "2.x",
		"1":      "1.x",
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
