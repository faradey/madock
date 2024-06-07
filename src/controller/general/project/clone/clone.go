package clone

import (
	"github.com/faradey/madock/src/helper/cli/arg_struct"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	"github.com/faradey/madock/src/helper/paths"
	"strings"
)

func Execute() {
	args := attr.Parse(new(arg_struct.ControllerGeneralProjectClone)).(*arg_struct.ControllerGeneralProjectClone)

	cloneName := args.Name
	if strings.Contains(cloneName, ".") || strings.Contains(cloneName, " ") {
		fmtc.ErrorLn("The project folder name cannot contain a period or space")
		return
	}
	projectsPath := paths.GetExecDirPath() + "/projects"
	dirs := paths.GetDirs(projectsPath)
	for _, val := range dirs {
		if val == cloneName {
			fmtc.ErrorLn("The project with the same name is exist")
			return
		}
	}

}
