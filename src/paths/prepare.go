package paths

import (
	"log"
	"os"
	"strings"
)

func PrepareDirsForProject() {
	projectName := GetRunDirName()
	execDir := GetExecDirPath()
	projectPath := execDir + "/projects/" + projectName

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

func MakeDirsByPath(val string) {
	val = strings.Trim(val, "/")
	if val != "" {
		dirs := strings.Split(val, "/")
		for i := 0; i < len(dirs); i++ {
			if _, err := os.Stat("/" + strings.Join(dirs[:i+1], "/")); os.IsNotExist(err) {
				err = os.Mkdir("/"+strings.Join(dirs[:i+1], "/"), 0755)
				if err != nil {
					log.Fatal(err)
				}
			}
		}
	}
}
