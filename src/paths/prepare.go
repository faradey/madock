package paths

import (
	"log"
	"os"
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
