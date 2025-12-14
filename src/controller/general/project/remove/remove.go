package remove

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/docker"
	"github.com/faradey/madock/src/helper/logger"
	"github.com/faradey/madock/src/helper/paths"
	"github.com/faradey/madock/src/helper/ports"
)

type ArgsStruct struct {
	attr.Arguments
	Force bool   `arg:"-f,--force" help:"Skip interactive confirmations (requires --name)"`
	Name  string `arg:"-n,--name" help:"Project name to remove (required with --force)"`
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

	// Remove port allocations for this project
	ports.GetRegistry().RemoveProject(projectName)

	fmtc.SuccessLn("Project was removed successfully")
	fmtc.SuccessLn("!!! Close the terminal for the changes to take effect !!!")
}
