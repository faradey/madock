package restore

import (
	"bufio"
	"compress/gzip"
	"fmt"

	"github.com/faradey/madock/src/command"
	"github.com/faradey/madock/src/controller/general/rebuild"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/docker"
	"github.com/faradey/madock/src/helper/logger"
	"github.com/faradey/madock/src/helper/paths"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"snapshot:restore"},
		Handler:  Execute,
		Help:     "Restore snapshot",
		Category: "snapshot",
	})
}

func Execute() {
	projectName := configs.GetProjectName()
	projectConf := configs.GetCurrentProjectConfig()

	dbsPath := paths.GetExecDirPath() + "/projects/" + projectName + "/backup/snapshot"
	var snapshotNames []string
	if paths.IsFileExist(dbsPath) {
		snapshotNames = paths.GetDirs(dbsPath)
		if len(snapshotNames) == 0 {
			fmt.Println("No snapshots")
		}
		for index, snapshotName := range snapshotNames {
			fmt.Println(strconv.Itoa(index+1) + ") " + filepath.Base(snapshotName))
		}
	}

	if len(snapshotNames) == 0 {
		logger.Fatal("No snapshots found for restore")
	}

	fmt.Println("Choose one of the offered variants")
	buf := bufio.NewReader(os.Stdin)
	sentence, err := buf.ReadBytes('\n')
	selected := strings.TrimSpace(string(sentence))
	selectedInt := 0
	if err != nil {
		logger.Fatalln(err)
	} else {
		selectedInt, err = strconv.Atoi(selected)

		if err != nil || selectedInt < 1 || selectedInt > len(snapshotNames) {
			logger.Fatal("The item you selected was not found")
		}
	}
	RestoreSnapshot(projectName, projectConf, selectedInt, snapshotNames, dbsPath)
	os.Args = append(os.Args, "-c")
	rebuild.Execute()
	fmt.Println("Snapshot restored successfully")
}

func RestoreSnapshot(projectName string, projectConf map[string]string, selectedInt int, snapshotNames []string, dbsPath string) {
	containerName := docker.GetContainerName(projectConf, projectName, "snapshot")
	docker.Down(projectName, false)
	docker.UpSnapshot(projectName)
	if paths.IsFileExist(dbsPath + "/" + snapshotNames[selectedInt-1] + "/db.tar.gz") {
		selectedFile, err := os.Open(dbsPath + "/" + snapshotNames[selectedInt-1] + "/db.tar.gz")
		if err != nil {
			logger.Fatal(err)
		}
		defer selectedFile.Close()
		cmd := exec.Command("docker", "exec", "-i", "-u", "root", containerName, "bash", "-c", "rm -rf /var/www/mysql/* && cd /var/www/mysql && tar -zxf -")
		out, err := gzip.NewReader(selectedFile)
		if err != nil {
			logger.Fatal(err)
		}
		defer out.Close()
		cmd.Stdin = out
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			logger.Fatal(err, containerName)
		}
	}

	if projectConf["db2/enabled"] == "true" && paths.IsFileExist(dbsPath+"/"+snapshotNames[selectedInt-1]+"/db2.tar.gz") {
		selectedFileDb2, err := os.Open(dbsPath + "/" + snapshotNames[selectedInt-1] + "/db2.tar.gz")
		if err != nil {
			logger.Fatal(err)
		}
		defer selectedFileDb2.Close()
		cmd := exec.Command("docker", "exec", "-i", "-u", "root", containerName, "bash", "-c", "rm -rf /var/www/mysql2/mysql/* && cd /var/www/mysql2/mysql && tar -zxf -")
		outDb2, err := gzip.NewReader(selectedFileDb2)
		if err != nil {
			logger.Fatal(err)
		}
		defer outDb2.Close()
		cmd.Stdin = outDb2
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			logger.Fatal(err, containerName)
		}
	}

	if paths.IsFileExist(dbsPath + "/" + snapshotNames[selectedInt-1] + "/files.tar.gz") {
		selectedFileFiles, err := os.Open(dbsPath + "/" + snapshotNames[selectedInt-1] + "/files.tar.gz")
		if err != nil {
			logger.Fatal(err)
		}
		defer selectedFileFiles.Close()
		cmd := exec.Command("docker", "exec", "-i", "-u", "root", containerName, "bash", "-c", "rm -rf /var/www/html/* && cd /var/www/html && tar -zxf -")
		outFiles, err := gzip.NewReader(selectedFileFiles)
		if err != nil {
			logger.Fatal(err)
		}
		defer outFiles.Close()
		cmd.Stdin = outFiles
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			logger.Fatal(err, containerName)
		}
	}

	docker.StopSnapshot(projectName)
}
