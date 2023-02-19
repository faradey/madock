package paths

import (
	"log"
	"os"
	"strings"
)

func PrepareDirsForProject() {
	projectName := GetProjectName()
	projectPath := GetExecDirPath() + "/projects/" + projectName
	MakeDirsByPath(projectPath)
	MakeDirsByPath(projectPath + "/docker")
	MakeDirsByPath(projectPath + "/docker/nginx")
}

func MakeDirsByPath(val string) string {
	trimVal := strings.Trim(val, "/")
	if trimVal != "" {
		dirs := strings.Split(trimVal, "/")
		for i := 0; i < len(dirs); i++ {
			if _, err := os.Stat("/" + strings.Join(dirs[:i+1], "/")); os.IsNotExist(err) {
				err = os.Mkdir("/"+strings.Join(dirs[:i+1], "/"), 0755)
				if err != nil {
					log.Fatal(err)
				}
			}
		}
	}

	return val
}
