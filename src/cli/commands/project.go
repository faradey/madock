package commands

import (
	"bufio"
	"fmt"
	"github.com/faradey/madock/src/cli/fmtc"
	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/docker/builder"
	"github.com/faradey/madock/src/paths"
	"log"
	"os"
	"strings"
)

func ProjectRemove() {
	fmt.Println("Are you sure? (y/n)")
	fmt.Print("> ")
	buf := bufio.NewReader(os.Stdin)
	sentence, err := buf.ReadBytes('\n')
	if err != nil {
		log.Fatalln(err)
	}
	result := strings.ToLower(strings.TrimSpace(string(sentence)))
	if result == "y" && len(configs.GetProjectName()) > 0 {
		builder.Down(true)
		err := os.RemoveAll(paths.GetExecDirPath() + "/projects/" + configs.GetProjectName() + "/")
		if err != nil {
			log.Fatal(err)
		}

		err = os.RemoveAll(paths.GetExecDirPath() + "/aruntime/projects/" + configs.GetProjectName() + "/")
		if err != nil {
			log.Fatal(err)
		}

		err = os.RemoveAll(paths.GetRunDirPath())
		if err != nil {
			log.Fatal(err)
		}
		fmtc.SuccessLn("Project was removed successfully")
	}
}
