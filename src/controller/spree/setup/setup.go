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
	"github.com/faradey/madock/v3/src/model/versions/spree"
	setupreg "github.com/faradey/madock/v3/src/setup"
)

type Handler struct{}

func (h *Handler) Execute(ctx *setupreg.SetupContext) {
	Execute(ctx.ProjectName, ctx.ProjectConf, ctx.ContinueSetup, ctx.Args, ctx.DetectedVersion)
}

func init() {
	setupreg.Register(setupreg.PlatformInfo{
		Name:        "spree",
		DisplayName: "Spree Commerce",
		Language:    "ruby",
		Order:       50,
	}, &Handler{})
}

func Execute(projectName string, projectConf map[string]string, continueSetup bool, args *arg_struct.ControllerGeneralSetup, detectedVersion string) {
	toolsDefVersions := spree.GetVersions()
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
		fmtc.InfoIconLn(fmt.Sprintf("Using detected Spree version: %s", detectedVersion))
	}

	if !usePreset && detectedVersion == "" && args.PlatformVersion == "" && continueSetup {
		presets := preset.GetSpreePresets()
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
	toolsDefVersions.Platform = "spree"
	toolsDefVersions.Language = "ruby"
	toolsDefVersions.DbType = "PostgreSQL"

	fmt.Println("")
	if usePreset {
		fmtc.Title("Spree version: " + toolsDefVersions.PlatformVersion)
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

	// Containers up first so git clone runs inside the ruby /
	// storefront containers, not on the host. Entrypoints poll for
	// project files so an empty workdir at boot is safe.
	if args.Download || args.Install || continueSetup {
		rebuild.Execute()
	}

	if args.Download {
		DownloadSpree(projectName)
	}

	if args.Install {
		install.Spree(projectName, toolsDefVersions.PlatformVersion)
	}
}

// DownloadSpree clones the upstream spree/spree_starter (Rails admin)
// via the ruby container, then clones spree/storefront via the
// storefront (nodejs) container when storefront is enabled.
func DownloadSpree(projectName string) {
	target := paths.GetRunDirPath()
	if !isDirEmpty(target) {
		fmtc.WarningLn("Skipping download — project directory is not empty: " + target)
		return
	}
	projectConf := configs.GetCurrentProjectConfig()
	repo := "https://github.com/spree/spree_starter.git"
	fmtc.InfoIconLn("Cloning " + repo + " into " + target)
	stage := "download-spree123456789"
	mainScript := "rm -rf ./" + stage +
		" && git clone --depth 1 " + repo + " ./" + stage +
		" && shopt -s dotglob" +
		" && mv ./" + stage + "/* ./ 2>/dev/null || true" +
		" && rm -rf ./" + stage
	runSpreeInContainer(projectConf, projectName, "ruby", "ruby", mainScript)

	if projectConf["spree/storefront/enabled"] != "true" {
		return
	}
	storefrontGitURL := projectConf["spree/storefront/git_url"]
	if storefrontGitURL == "" {
		storefrontGitURL = "https://github.com/spree/storefront.git"
	}
	storefrontDir := projectConf["spree/storefront/path"]
	if storefrontDir == "" {
		storefrontDir = "storefront"
	}
	storefrontTarget := target + "/" + storefrontDir
	if _, err := os.Stat(storefrontTarget); err == nil {
		fmtc.WarningLn("Skipping storefront download — " + storefrontTarget + " already exists.")
		return
	}
	fmtc.InfoIconLn("Cloning " + storefrontGitURL + " into " + storefrontTarget)
	// Storefront clone goes alongside the rails app — run as the ruby
	// user so the storefront subdir is owned by the same project user.
	storefrontScript := "git clone --depth 1 " + storefrontGitURL + " " + storefrontDir
	runSpreeInContainer(projectConf, projectName, "ruby", "ruby", storefrontScript)
}

func runSpreeInContainer(projectConf map[string]string, projectName, serviceHint, userHint, script string) {
	service, user, workdir := cli.GetEnvForUserServiceWorkdir(serviceHint, userHint, projectConf["workdir"])
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
		fmtc.WarningLn("Download step failed: " + err.Error())
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
	presets := preset.GetSpreePresets()

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
		"latest": "5",
		"stable": "4.10",
		"5":      "5",
		"4":      "4.10",
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
