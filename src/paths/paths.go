package paths

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/faradey/madock/src/helper"
)

func GetExecDirPath() string {
	var dirAbsPath string

	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exReal, err := filepath.EvalSymlinks(ex)
	if err != nil {
		dirAbsPath = filepath.Dir(ex)
		return dirAbsPath
	} else {
		dirAbsPath = filepath.Dir(exReal)
		return dirAbsPath
	}

	panic("Unknown error")
}

func GetExecDirName() string {
	return filepath.Base(GetExecDirPath())
}

func GetExecDirNameByPath(path string) string {
	return filepath.Base(path)
}

func GetRunDirPath() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return ""
	}

	return dir
}

func GetRunDirName() string {
	return filepath.Base(GetRunDirPath())
}

func GetRunDirNameWithHash() string {
	return filepath.Base(GetRunDirPath()) + "__" + strconv.Itoa(int(helper.Hash(GetRunDirPath())))
}

func GetDirs(path string) (dirs []string) {
	items, err := os.ReadDir(path)
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range items {
		if file.IsDir() {
			dirs = append(dirs, file.Name())
		}
	}

	return dirs
}

func GetFiles(path string) (dirs []string) {
	items, err := os.ReadDir(path)
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range items {
		if !file.IsDir() {
			dirs = append(dirs, file.Name())
		}
	}

	return dirs
}

func GetFilesRecursively(path string) (dirs []string) {
	items, err := os.ReadDir(path)
	if err == nil {
		for _, file := range items {
			if !file.IsDir() {
				dirs = append(dirs, path+"/"+file.Name())
			} else {
				dirs = append(dirs, GetFilesRecursively(path+"/"+file.Name())...)
			}
		}
	}

	return dirs
}

func GetDBFiles(path string) (dirs []string) {
	items, err := os.ReadDir(path)
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range items {
		if !file.IsDir() {
			if file.Name()[0:1] != "." &&
				strings.Contains(strings.ToLower(file.Name()), ".sql") &&
				!strings.Contains(strings.ToLower(path), "/dev/tests/acceptance") &&
				!strings.Contains(strings.ToLower(path), strings.ToLower(strings.Trim(GetRunDirPath(), "/"))+"/vendor/") {
				dirs = append(dirs, path+"/"+file.Name())
			}
		} else {
			dirs = append(dirs, GetDBFiles(path+"/"+file.Name())...)
		}
	}

	return dirs
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

func GetActiveProjects() []string {
	var activeProjects []string
	projects := GetDirs(GetExecDirPath() + "/aruntime/projects")
	for _, projectName := range projects {
		if _, err := os.Stat(GetExecDirPath() + "/aruntime/projects/" + projectName + "/docker-compose.yml"); !os.IsNotExist(err) {
			duration := time.Millisecond * 20
			time.Sleep(duration)
			cmd := exec.Command("docker", "compose", "-f", GetExecDirPath()+"/aruntime/projects/"+projectName+"/docker-compose.yml", "ps", "--format", "json")
			result, err := cmd.CombinedOutput()
			if err != nil {
				log.Fatal(err)
			}
			if len(result) > 100 {
				activeProjects = append(activeProjects, projectName)
			}
		}
	}
	return activeProjects
}
