package remove

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/faradey/madock/v3/src/command"
	"github.com/faradey/madock/v3/src/helper/cli/attr"
	"github.com/faradey/madock/v3/src/helper/cli/fmtc"
	"github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/configs/aruntime/nginx"
	"github.com/faradey/madock/v3/src/helper/docker"
	"github.com/faradey/madock/v3/src/helper/logger"
	"github.com/faradey/madock/v3/src/helper/paths"
	"github.com/faradey/madock/v3/src/helper/ports"
)

type ArgsStruct struct {
	attr.Arguments
	Force bool   `arg:"-f,--force" help:"Skip interactive confirmations (requires --name)"`
	Name  string `arg:"-n,--name" help:"Project name to remove (required with --force)"`
}

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"project:remove"},
		Handler:  Execute,
		Help:     "Remove project",
		Category: "project",
		ArgsType: new(ArgsStruct),
	})
}

func Execute() {
	args := attr.Parse(new(ArgsStruct)).(*ArgsStruct)

	projectName := configs.GetProjectName()

	// Non-interactive mode with --force flag
	if args.Force {
		if args.Name == "" {
			fmtc.ErrorLn("--force requires --name to specify the project name")
			return
		}
		if args.Name != projectName {
			fmtc.ErrorLn("Project name mismatch. Current project: " + projectName + ", specified: " + args.Name)
			return
		}
		removeProject(projectName)
		return
	}

	// Interactive mode
	fmt.Println("Are you sure? (y/n)")
	fmt.Print("> ")
	buf := bufio.NewReader(os.Stdin)
	sentence, err := buf.ReadBytes('\n')
	if err != nil {
		logger.Fatalln(err)
	}
	result := strings.ToLower(strings.TrimSpace(string(sentence)))
	if result == "y" && len(projectName) > 0 {
		pp := paths.NewProjectPaths(projectName)
		fmt.Println("The following items will be removed:")
		fmt.Println(paths.GetExecDirPath() + "/projects/" + projectName + "/")
		fmt.Println(pp.RuntimeDir())
		fmt.Println(paths.GetRunDirPath())
		fmt.Println("Containers, images and volumes associated with the project.")
		fmt.Println("")
		fmt.Println("Enter the project name \"" + projectName + "\" to confirm the deletion of the project")
		fmt.Print("> ")
		buf = bufio.NewReader(os.Stdin)
		sentence, err = buf.ReadBytes('\n')
		if err != nil {
			logger.Fatalln(err)
		}
		result = strings.TrimSpace(string(sentence))
		if result == projectName {
			removeProject(projectName)
		} else {
			fmtc.WarningLn("The project was not removed. The entered value does not match the project name.")
		}
	}
}

func removeProject(projectName string) {
	docker.Down(projectName, true)

	pp := paths.NewProjectPaths(projectName)
	err := os.RemoveAll(paths.GetExecDirPath() + "/projects/" + projectName + "/")
	if err != nil {
		logger.Fatal(err)
	}

	err = os.RemoveAll(pp.RuntimeDir())
	if err != nil {
		logger.Fatal(err)
	}

	err = os.RemoveAll(paths.GetRunDirPath())
	if err != nil {
		logger.Fatal(err)
	}

	// Take the project out of the shared proxy.
	//
	// Removal is the one moment this belongs to. A stopped project keeps its
	// routing on purpose — it is coming back, and rewriting every other
	// project's configuration for a pause would be churn and risk for nothing.
	// A removed project is not coming back, and its server block points at a
	// container that no longer exists.
	//
	// Nothing did this before, so the block survived until something else
	// happened to regenerate the file.
	// Regenerated on behalf of a project that still exists, never the removed
	// one: MakeConf recreates directories and allocates ports for the name it is
	// given, so naming the corpse brings its registry entry and its port
	// reservation straight back.
	remaining := paths.GetActiveProjects()
	if len(remaining) > 0 {
		nginx.MakeConf(remaining[0])
		if err := docker.ReloadNginx(); err != nil {
			logger.Println(err)
		}
	}

	// Ports come last, and the order is not cosmetic: generating the proxy
	// configuration allocates a port for the project it is given, so releasing
	// them first only meant handing them straight back. The removed project then
	// kept a reservation no other project could use.
	ports.GetRegistry().RemoveProject(projectName)

	fmtc.SuccessLn("Project was removed successfully")
	fmtc.SuccessLn("!!! Close the terminal for the changes to take effect !!!")
}
