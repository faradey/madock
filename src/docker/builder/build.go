package builder

import (
	"fmt"
	"github.com/faradey/madock/src/paths"
)

func Build() {
	buildNginx()
}

func buildNginx() {
	projectsNames := paths.GetDirs(paths.GetExecDirPath() + "/projects")

	fmt.Println(projectsNames)
}
