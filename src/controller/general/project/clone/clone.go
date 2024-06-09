package clone

import (
	"compress/gzip"
	"github.com/faradey/madock/src/controller/general/snapshot/create"
	"github.com/faradey/madock/src/helper/cli/arg_struct"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/docker"
	"github.com/faradey/madock/src/helper/logger"
	"github.com/faradey/madock/src/helper/paths"
	"os"
	"os/exec"
	"path/filepath"
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

	projectConf := configs.GetCurrentProjectConfig()
	exPath := paths.GetExecDirPath()
	projectName := configs.GetProjectName()
	currentDest := paths.MakeDirsByPath(exPath + "/projects/" + projectName + "/")
	dest := paths.MakeDirsByPath(exPath + "/projects/" + cloneName + "/")
	configs.PrepareDirsForProject(cloneName)
	files := paths.GetFilesRecursively(currentDest)
	for _, val := range files {
		paths.MakeDirsByPath(dest + strings.Replace(strings.Replace(val, currentDest, "", -1), "/"+filepath.Base(val), "", -1))
		err := paths.Copy(val, dest+strings.Replace(val, currentDest, "", -1))
		if err != nil {
			logger.Fatal(err)
		}
	}

	clonePathParts := strings.Split(projectConf["path"], "/")
	clonePath := strings.Join(clonePathParts[:len(clonePathParts)-1], "/") + "/" + cloneName + "/"
	configs.SetParam(cloneName, "path", clonePath, projectConf["activeScope"], "")
	cloneProjectConf := configs.GetProjectConfig(cloneName)
	if projectConf["platform"] != "pwa" {
		create.GetDB(projectConf, projectName, dest)
	}
	create.GetFiles(projectConf, projectName, dest)
	containerName := docker.GetContainerName(cloneProjectConf, cloneName, "snapshot")
	docker.UpSnapshot(cloneName)
	if paths.IsFileExist(dest + "db.tar.gz") {
		selectedFile, err := os.Open(dest + "db.tar.gz")
		if err != nil {
			logger.Fatal(err)
		}
		defer selectedFile.Close()
		cmd := exec.Command("docker", "exec", "-i", "-u", "root", containerName, "bash", "-c", "rm -rf /var/www/mysql/* && cd /var/www/mysql && tar -zxf -")
		out, err := gzip.NewReader(selectedFile)
		if err != nil {
			logger.Fatal(err)
		}
		cmd.Stdin = out
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			logger.Fatal(err, containerName)
		}
		err = os.Remove(dest + "db.tar.gz")
		if err != nil {
			logger.Fatal(err)
		}
	}

	if paths.IsFileExist(dest + "db2.tar.gz") {
		selectedFileDb2, err := os.Open(dest + "db2.tar.gz")
		if err != nil {
			logger.Fatal(err)
		}
		defer selectedFileDb2.Close()
		cmd := exec.Command("docker", "exec", "-i", "-u", "root", containerName, "bash", "-c", "rm -rf /var/www/mysql2/mysql/* && cd /var/www/mysql2/mysql && tar -zxf -")
		out, err := gzip.NewReader(selectedFileDb2)
		if err != nil {
			logger.Fatal(err)
		}
		cmd.Stdin = out
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			logger.Fatal(err, containerName)
		}
		err = os.Remove(dest + "db2.tar.gz")
		if err != nil {
			logger.Fatal(err)
		}
	}

	if paths.IsFileExist(dest + "files.tar.gz") {
		selectedFileFiles, err := os.Open(dest + "files.tar.gz")
		if err != nil {
			logger.Fatal(err)
		}
		defer selectedFileFiles.Close()
		cmd := exec.Command("docker", "exec", "-i", "-u", "root", containerName, "bash", "-c", "rm -rf /var/www/html/* && cd /var/www/html && tar -zxf -")
		out, err := gzip.NewReader(selectedFileFiles)
		if err != nil {
			logger.Fatal(err)
		}
		cmd.Stdin = out
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			logger.Fatal(err, containerName)
		}
		err = os.Remove(dest + "files.tar.gz")
		if err != nil {
			logger.Fatal(err)
		}
	}

	docker.StopSnapshot(cloneName)

	fmtc.SuccessLn("Project cloned successfully.\nThe new project name is " + cloneName + ".\nThe path to the project is " + clonePath)
}
