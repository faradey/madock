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

	// Containers must be up before Download so all platform-specific
	// scaffolding (git clone, composer init) runs inside the project
	// containers — the host needs only `docker`. Mirrors Magento's
	// order: rebuild → DownloadX (docker exec) → install.
	if args.Download || args.Install || continueSetup {
		rebuild.Execute()
	}

	if args.Download {
		DownloadBigcommerce(projectName, presetCode)
	}

	if args.Install {
		install.Bigcommerce(projectName, presetCode)
	}
}

// DownloadBigcommerce scaffolds the project inside the project
// container based on the selected preset. Catalyst / Stencil /
// app-node clone via git in the nodejs container; api-php writes a
// minimal composer.json via the php container. Matches Magento's
// "docker exec into the workdir" pattern so madock has no host-side
// dependency on git / composer / npm.
func DownloadBigcommerce(projectName, presetCode string) {
	target := paths.GetRunDirPath()
	if !isDirEmpty(target) {
		fmtc.WarningLn("Skipping download — project directory is not empty: " + target)
		return
	}
	projectConf := configs.GetCurrentProjectConfig()

	var serviceName, userName, repoURL, label string
	switch presetCode {
	case "catalyst":
		serviceName, userName = "nodejs", "node"
		repoURL = "https://github.com/bigcommerce/catalyst.git"
		label = "BigCommerce Catalyst storefront"
	case "stencil":
		serviceName, userName = "nodejs", "node"
		repoURL = "https://github.com/bigcommerce/cornerstone.git"
		label = "BigCommerce Cornerstone theme"
	case "app-node":
		serviceName, userName = "nodejs", "node"
		repoURL = "https://github.com/bigcommerce/sample-app-nodejs.git"
		label = "BigCommerce sample Node app"
	default:
		// api-php
		fmtc.InfoIconLn("Initialising bigcommerce/api project in " + target)
		runInContainer(projectConf, projectName, "php", "www-data",
			"composer init --no-interaction "+
				"--name=bigcommerce/api-project "+
				"--type=project "+
				"--require=bigcommerce/api:^3.3 "+
				"--stability=stable")
		return
	}

	fmtc.InfoIconLn("Cloning " + label + " into " + target)
	// Clone into a temp subdir then move contents, mirroring
	// Magento's Download trick — `git clone .` refuses a non-empty
	// dir, and the workdir may already hold .madock / .docker
	// artefacts from rebuild.
	stage := "download-bigcommerce123456789"
	script := "rm -rf ./" + stage +
		" && git clone --depth 1 " + repoURL + " ./" + stage +
		" && shopt -s dotglob" +
		" && mv ./" + stage + "/* ./ 2>/dev/null || true" +
		" && rm -rf ./" + stage
	runInContainer(projectConf, projectName, serviceName, userName, script)
}

// runInContainer is a thin wrapper around `docker exec -u user
// container bash -c script` that keeps the BigCommerce Download flow
// readable.
func runInContainer(projectConf map[string]string, projectName, serviceHint, userHint, script string) {
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
