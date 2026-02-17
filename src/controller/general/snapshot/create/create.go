package create

import (
	"compress/gzip"
	"fmt"

	"github.com/faradey/madock/src/command"
	"github.com/faradey/madock/src/helper/cli/arg_struct"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/docker"
	"github.com/faradey/madock/src/helper/logger"
	"github.com/faradey/madock/src/helper/paths"
	"os"
	"time"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"snapshot:create"},
		Handler:  Execute,
		Help:     "Create snapshot",
		Category: "snapshot",
	})
}

func Execute() {
	args := attr.Parse(new(arg_struct.ControllerGeneralSnapshotCreate)).(*arg_struct.ControllerGeneralSnapshotCreate)
	projectConf := configs.GetCurrentProjectConfig()
	exPath := paths.GetExecDirPath()
	projectName := configs.GetProjectName()
	dest := paths.MakeDirsByPath(exPath + "/projects/" + projectName + "/backup/snapshot")

	name := "snapshot-"
	if args.Name != "" {
		name += args.Name + "-"
	}
	name += time.Now().Format("2006-01-02-15-04-05")

	dbsPath := paths.MakeDirsByPath(dest + "/" + name + "/")
	GetDB(projectConf, projectName, dbsPath)
	GetFiles(projectConf, projectName, dbsPath)
	fmt.Println("Snapshot completed successfully")
}

func GetDB(projectConf map[string]string, projectName string, dbsPath string) {
	selectedFile, err := os.Create(dbsPath + "db.tar.gz")
	if err != nil {
		logger.Fatal(err)
	}
	defer selectedFile.Close()
	writer := gzip.NewWriter(selectedFile)
	defer writer.Close()
	cmd, prepErr := docker.PrepareContainerExec(docker.GetContainerName(projectConf, projectName, "db"), "root", false, "bash", "-c", "cd /var/lib/mysql && tar -czf /tmp/db.tar.gz . && cat /tmp/db.tar.gz")
	if prepErr != nil {
		logger.Fatal(prepErr)
	}
	cmd.Stdout = writer
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		logger.Fatal(err)
	}

	if projectConf["db2/enabled"] == "true" {
		selectedFileDb2, err := os.Create(dbsPath + "db2.tar.gz")
		if err != nil {
			logger.Fatal(err)
		}
		defer selectedFileDb2.Close()
		writerDb2 := gzip.NewWriter(selectedFileDb2)
		defer writerDb2.Close()
		cmd, prepErr = docker.PrepareContainerExec(docker.GetContainerName(projectConf, projectName, "db2"), "root", false, "bash", "-c", "cd /var/lib/mysql && tar -czf /tmp/db2.tar.gz . && cat /tmp/db2.tar.gz")
		if prepErr != nil {
			logger.Fatal(prepErr)
		}
		cmd.Stdout = writerDb2
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			logger.Fatal(err)
		}
	}
}

func GetFiles(projectConf map[string]string, projectName string, dbsPath string) {
	selectedFileFiles, err := os.Create(dbsPath + "files.tar.gz")
	if err != nil {
		logger.Fatal(err)
	}
	defer selectedFileFiles.Close()
	writerFiles := gzip.NewWriter(selectedFileFiles)
	defer writerFiles.Close()
	cmd, prepErr := docker.PrepareContainerExec(docker.GetContainerName(projectConf, projectName, "php"), "root", false, "bash", "-c", "cd /var/www/html && tar -czf /tmp/files.tar.gz . && cat /tmp/files.tar.gz")
	if prepErr != nil {
		logger.Fatal(prepErr)
	}
	cmd.Stdout = writerFiles
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		logger.Fatal(err)
	}
}
