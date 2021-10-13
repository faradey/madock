package paths

import (
	"log"
	"os"
	"strings"
)

func PrepareDirsForProject() {
	projectName := GetRunDirName()
	projectPath := GetExecDirPath() + "/projects/" + projectName

	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		err = os.Mkdir(projectPath, 0755)
		if err != nil {
			log.Fatal(err)
		}
	}

	checkPath := projectPath + "/docker"
	if _, err := os.Stat(checkPath); os.IsNotExist(err) {
		err = os.Mkdir(checkPath, 0755)
		if err != nil {
			log.Fatal(err)
		}
	}

	checkPath = projectPath + "/docker/nginx"
	if _, err := os.Stat(checkPath); os.IsNotExist(err) {
		err = os.Mkdir(checkPath, 0755)
		if err != nil {
			log.Fatal(err)
		}
	}
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
