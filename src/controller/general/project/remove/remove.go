package remove

import (
	"bufio"
	"fmt"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/docker"
	"github.com/faradey/madock/src/helper/logger"
	"github.com/faradey/madock/src/helper/paths"
	"os"
	"strings"
)

type ArgsStruct struct {
	attr.Arguments
}

func Execute() {
	attr.Parse(new(ArgsStruct))

	fmt.Println("Are you sure? (y/n)")
	fmt.Print("> ")
	buf := bufio.NewReader(os.Stdin)
	sentence, err := buf.ReadBytes('\n')
	if err != nil {
		logger.Fatalln(err)
	}
	result := strings.ToLower(strings.TrimSpace(string(sentence)))
	projectName := configs.GetProjectName()
	if result == "y" && len(projectName) > 0 {
		fmt.Println("The following items will be removed:")
		fmt.Println(paths.GetExecDirPath() + "/projects/" + projectName + "/")
		fmt.Println(paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/")
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
			docker.Down(projectName, true)
			err := os.RemoveAll(paths.GetExecDirPath() + "/projects/" + projectName + "/")
			if err != nil {
				logger.Fatal(err)
			}

			err = os.RemoveAll(paths.GetExecDirPath() + "/aruntime/projects/" + projectName + "/")
			if err != nil {
				logger.Fatal(err)
			}

			err = os.RemoveAll(paths.GetRunDirPath())
			if err != nil {
				logger.Fatal(err)
			}
			fmtc.SuccessLn("Project was removed successfully")
			fmtc.SuccessLn("!!! Close the terminal for the changes to take effect !!!")
		} else {
			fmtc.WarningLn("The project was not removed. The entered value does not match the project name.")
		}
	}
}
