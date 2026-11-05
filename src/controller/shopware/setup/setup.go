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
	configs2 "github.com/faradey/madock/v4/src/helper/configs"
	"github.com/faradey/madock/v4/src/helper/configs/projects"
	"github.com/faradey/madock/v4/src/helper/docker"
	"github.com/faradey/madock/v4/src/helper/logger"
	"github.com/faradey/madock/v4/src/helper/paths"
	"github.com/faradey/madock/v4/src/helper/setup/tools"
	"github.com/faradey/madock/v4/src/model/versions/shopware"
	setupreg "github.com/faradey/madock/v4/src/setup"
)

type Handler struct{}

func (h *Handler) Execute(ctx *setupreg.SetupContext) {
	Execute(ctx.ProjectName, ctx.ProjectConf, ctx.ContinueSetup, ctx.Args)
}

func init() {
	setupreg.Register(setupreg.PlatformInfo{
		Name:        "shopware",
		DisplayName: "Shopware",
		Language:    "php",
		Order:       40,
	}, &Handler{})
}

func Execute(projectName string, projectConf map[string]string, continueSetup bool, args *arg_struct.ControllerGeneralSetup) {
	toolsDefVersions := shopware.GetVersions("")

	platformVersion := ""
	if args.PlatformVersion != "" {
		platformVersion = args.PlatformVersion
		// Read the version's own defaults here rather than leaving it to the
		// branch below, which only runs while Php is still empty — so naming a
		// version *and* a PHP version skipped the table entirely and left
		// Composer, the database and the search engine at the values for an
		// unknown version, which are empty strings. Magento had the same hole
		// on the same flag, where it meant Composer 1 and an install that could
		// not resolve a package.
		//
		// Explicit flags still win: each is applied over this below.
		toolsDefVersions = shopware.GetVersions(platformVersion)
		if args.Php != "" {
			toolsDefVersions.Php = args.Php
		}
	}

	if continueSetup {
		if toolsDefVersions.Php == "" {
			if platformVersion == "" {
				platformVersion = tools.RequireAnswer("Specify Shopware version", "--platform-version")
			}
			if platformVersion != "" {
				toolsDefVersions = shopware.GetVersions(platformVersion)
			} else {
				Execute(projectName, projectConf, continueSetup, args)
				return
			}
		}

		tools.PopulateFromConfig(&toolsDefVersions, projectConf)

		fmt.Println("")
		fmtc.Title("Your Shopware version is " + toolsDefVersions.PlatformVersion)

		if args.Php == "" {
			tools.Php(&toolsDefVersions.Php)
		} else {
			toolsDefVersions.Php = args.Php
		}
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
		if args.Composer == "" {
			tools.Composer(&toolsDefVersions.Composer)
		} else {
			toolsDefVersions.Composer = args.Composer
		}
		if args.SearchEngine == "" {
			tools.SearchEngine(&toolsDefVersions.SearchEngine)
		} else {
			toolsDefVersions.SearchEngine = args.SearchEngine
		}
		if strings.EqualFold(toolsDefVersions.SearchEngine, "Elasticsearch") {
			if args.SearchEngineVersion == "" {
				tools.Elastic(&toolsDefVersions.Elastic)
			} else {
				toolsDefVersions.Elastic = args.SearchEngineVersion
			}
		} else if strings.EqualFold(toolsDefVersions.SearchEngine, "OpenSearch") {
			if args.SearchEngineVersion == "" {
				tools.OpenSearch(&toolsDefVersions.OpenSearch)
			} else {
				toolsDefVersions.OpenSearch = args.SearchEngineVersion
			}
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
		if args.Hosts == "" {
			tools.Hosts(projectName, &toolsDefVersions.Hosts, projectConf)
		} else {
			toolsDefVersions.Hosts = args.Hosts
		}

		projects.SetEnvForProject(projectName, toolsDefVersions, configs2.GetProjectConfigOnly(projectName))
		paths.MakeDirsByPath(paths.GetExecDirPath() + "/projects/" + projectName + "/backup/db")

		fmtc.SuccessLn("\n" + "Finish set up environment")
		fmtc.ToDoLn("Optionally, you can configure SSH access to the development server in order ")
		fmtc.ToDoLn("to synchronize the database and media files. Enter SSH data in ")
		fmtc.ToDoLn(paths.GetExecDirPath() + "/projects/" + projectName + "/config.xml")
	}

	if args.Download || args.Install || continueSetup {
		rebuild.Execute()
	}

	if args.Download {
		DownloadShopware(projectName, platformVersion, args.SampleData)
	}

	if args.Install {
		install.Shopware(projectName, toolsDefVersions.PlatformVersion, args.SampleData)
	}
}

// createProjectScript is `composer create-project`, tried three times before the
// setup fails, as `pecl install` is in the php image.
//
// One failed download used to end a setup whose rebuild had just taken five
// minutes. Seen on three CI runs out of 25, 2026-11-01 to 2026-11-05: packagist
// answered in the same composer call, then "Failed to connect to api.github.com
// port 443 after 0 ms: Connection refused", and a new run minutes later passed.
// Who refused is not known. Nothing on the way to GitHub answers in 0 ms, which
// reads like a local address coming back from DNS, so a failed attempt prints what
// the name resolves to in this container. If that is a property of the machine
// the retry will not help, and those lines will say so.
//
// composer refuses a directory that is not empty, and a failed attempt can leave
// files in it, so each new attempt starts from an empty one.
func createProjectScript(version, dir string, pauseSeconds int) string {
	return fmt.Sprintf("(for attempt in 1 2 3; do "+
		"composer create-project shopware/production:%[1]s %[2]s && exit 0; "+
		"echo \"madock: composer create-project failed (attempt ${attempt} of 3). api.github.com resolves to:\"; "+
		"getent ahosts api.github.com 2>&1 || true; "+
		"rm -r -f %[2]s && mkdir %[2]s; "+
		"sleep %[3]d; "+
		"done; exit 1)", version, dir, pauseSeconds)
}

func DownloadShopware(projectName, version string, isSampleData bool) {
	projectConf := configs2.GetCurrentProjectConfig()
	sampleData := ""
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
		"cd " + workdir + " " +
			"&& rm -r -f " + workdir + "/download-shopware123456789 " +
			"&& mkdir " + workdir + "/download-shopware123456789 " +
			"&& " + createProjectScript(version, "./download-shopware123456789", 15) + " " +
			"&& shopt -s dotglob " +
			"&& mv  -v ./download-shopware123456789/* ./ " +
			"&& rm -r -f ./download-shopware123456789 " +
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
