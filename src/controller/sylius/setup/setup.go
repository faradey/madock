package setup

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/faradey/madock/v3/src/controller/general/install"
	"github.com/faradey/madock/v3/src/controller/general/rebuild"
	"github.com/faradey/madock/v3/src/helper/cli"
	"github.com/faradey/madock/v3/src/helper/cli/arg_struct"
	"github.com/faradey/madock/v3/src/helper/cli/fmtc"
	"github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/configs/projects"
	"github.com/faradey/madock/v3/src/helper/docker"
	"github.com/faradey/madock/v3/src/helper/paths"
	"github.com/faradey/madock/v3/src/helper/preset"
	"github.com/faradey/madock/v3/src/helper/setup/tools"
	"github.com/faradey/madock/v3/src/model/versions/sylius"
	setupreg "github.com/faradey/madock/v3/src/setup"
)

type Handler struct{}

func (h *Handler) Execute(ctx *setupreg.SetupContext) {
	Execute(ctx.ProjectName, ctx.ProjectConf, ctx.ContinueSetup, ctx.Args, ctx.DetectedVersion)
}

func init() {
	setupreg.Register(setupreg.PlatformInfo{
		Name:        "sylius",
		DisplayName: "Sylius",
		Language:    "php",
		Order:       55,
	}, &Handler{})
}

func Execute(projectName string, projectConf map[string]string, continueSetup bool, args *arg_struct.ControllerGeneralSetup, detectedVersion string) {
	toolsDefVersions := sylius.GetVersions("")
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
		toolsDefVersions = sylius.GetVersions(detectedVersion)
		fmtc.InfoIconLn(fmt.Sprintf("Using detected Sylius version: %s", detectedVersion))
	}

	if !usePreset && detectedVersion == "" && args.PlatformVersion == "" && continueSetup {
		presets := preset.GetSyliusPresets()
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
	toolsDefVersions.Platform = "sylius"
	toolsDefVersions.Language = "php"

	fmt.Println("")
	if usePreset {
		fmtc.Title("Sylius version: " + toolsDefVersions.PlatformVersion)
	}

	if !usePreset {
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

	// Containers up first so git clone runs inside the php container,
	// not on the host. php-fpm starts fine with an empty workdir, and
	// composer install (in the install handler) runs later once the
	// project tree is in place.
	if args.Download || args.Install || continueSetup {
		rebuild.Execute()
	}

	if args.Download {
		DownloadSylius(projectName)
	}

	if args.Install {
		install.Sylius(projectName, toolsDefVersions.PlatformVersion, args.SampleData)
	}
}

// DownloadSylius clones the upstream Sylius/Sylius-Standard repo into
// the current project root via the php container so the host has no
// git dependency. The standard project is a full Symfony app
// pre-configured with Sylius bundles, fixtures and the Webpack Encore
// frontend pipeline.
func DownloadSylius(projectName string) {
	target := paths.GetRunDirPath()
	if !isDirEmpty(target) {
		fmtc.WarningLn("Skipping download — project directory is not empty: " + target)
		return
	}
	projectConf := configs.GetCurrentProjectConfig()
	repo := "https://github.com/Sylius/Sylius-Standard.git"
	fmtc.InfoIconLn("Cloning " + repo + " into " + target)
	stage := "download-sylius123456789"
	script := "rm -rf ./" + stage +
		" && git clone --depth 1 " + repo + " ./" + stage +
		" && shopt -s dotglob" +
		" && mv ./" + stage + "/* ./ 2>/dev/null || true" +
		" && rm -rf ./" + stage
	service, user, workdir := cli.GetEnvForUserServiceWorkdir("php", "www-data", projectConf["workdir"])
	ttyFlag := "-i"
	if docker.IsTTYAvailable() {
		ttyFlag = "-it"
	}
	cmd := exec.Command("docker", "exec", ttyFlag, "-u", user,
		docker.GetContainerName(projectConf, projectName, service),
		"bash", "-c", "cd "+workdir+" && "+script)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmtc.WarningLn("Failed to clone Sylius-Standard: " + err.Error())
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

func findPresetByName(name string) *preset.Preset {
	name = strings.ToLower(name)
	presets := preset.GetSyliusPresets()

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
		"latest": "2",
		"stable": "1.13",
		"2":      "2",
		"1":      "1.13",
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
