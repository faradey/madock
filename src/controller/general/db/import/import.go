package _import

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/faradey/madock/src/helper/cli/arg_struct"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/configs/aruntime/project"
	"github.com/faradey/madock/src/helper/docker"
	"github.com/faradey/madock/src/helper/logger"
	"github.com/faradey/madock/src/helper/paths"
)

type progressReader struct {
	reader      io.Reader
	bytesRead   atomic.Int64
	totalBytes  int64
	lastPercent int
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	if n > 0 {
		pr.bytesRead.Add(int64(n))
	}
	return n, err
}

func (pr *progressReader) printProgress(done chan struct{}) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			fmt.Print("\r\033[K")
			return
		case <-ticker.C:
			read := pr.bytesRead.Load()
			percent := int(float64(read) / float64(pr.totalBytes) * 100)
			if percent > 100 {
				percent = 100
			}
			fmt.Printf("\rRestoring database... %d%% (%s / %s)", percent, formatBytes(read), formatBytes(pr.totalBytes))
		}
	}
}

func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1fGB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1fMB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1fKB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}

func Import() {
	projectConf := configs.GetCurrentProjectConfig()

	if projectConf["platform"] != "pwa" {
		args := attr.Parse(new(arg_struct.ControllerGeneralDbImport)).(*arg_struct.ControllerGeneralDbImport)

		option := ""
		if args.Force {
			option = "-f"
		}
		service := "db"
		if args.DBServiceName != "" {
			service = args.DBServiceName
		}

		user := "mysql"
		if args.User != "" {
			user = args.User
		}

		projectName := configs.GetProjectName()
		globalIndex := 0
		dbsPath := paths.GetExecDirPath() + "/projects/" + projectName + "/backup/db"
		var dbNames []string
		if paths.IsFileExist(dbsPath) {
			dbNames = paths.GetDBFiles(dbsPath)
			fmt.Println("Location: madock/projects/" + projectName + "/backup/db")
			if len(dbNames) == 0 {
				fmt.Println("No DB files")
			}
			for index, dbName := range dbNames {
				fmt.Println(strconv.Itoa(index+1) + ") " + filepath.Base(dbName))
				globalIndex += 1
			}
		}

		dbsPath = paths.GetRunDirPath()
		dbNames2 := paths.GetDBFiles(dbsPath)
		fmt.Println("Location: " + dbsPath)
		if len(dbNames2) == 0 {
			fmt.Println("No DB files")
		} else {
			dbNames = append(dbNames, dbNames2...)
		}
		for index, dbName := range dbNames2 {
			fmt.Println(strconv.Itoa(globalIndex+index+1) + ") " + filepath.Base(dbName) + "  " + dbName)
		}

		if len(dbNames) == 0 {
			logger.Fatal("No database files found for import")
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

			if err != nil || selectedInt < 1 || selectedInt > len(dbNames) {
				logger.Fatal("The item you selected was not found")
			}
		}

		filePath := dbNames[selectedInt-1]
		ext := strings.ToLower(filepath.Ext(filePath))

		selectedFile, err := os.Open(filePath)
		if err != nil {
			logger.Fatal(err)
		}
		defer selectedFile.Close()

		fileInfo, err := selectedFile.Stat()
		if err != nil {
			logger.Fatal(err)
		}
		totalSize := fileInfo.Size()

		containerName := docker.GetContainerName(projectConf, projectName, service)

		mysqlCommandName := "mysql"
		if projectConf["db/repository"] == "mariadb" && project.CompareVersions(projectConf["db/version"], "10.5") != -1 {
			mysqlCommandName = "mariadb"
		}

		var cmd *exec.Cmd
		cmdFKeys := exec.Command("docker", "exec", "-i", "-u", user, containerName, mysqlCommandName, "-u", "root", "-p"+projectConf["db/root_password"], "-h", service, "-f", "--execute", "SET FOREIGN_KEY_CHECKS=0;", projectConf["db/database"])
		if err := cmdFKeys.Run(); err != nil {
			logger.Fatalln("Failed to disable foreign key checks:", err)
		}
		if option != "" {
			cmd = exec.Command("docker", "exec", "-i", "-u", user, containerName, mysqlCommandName, option, "-u", "root", "-p"+projectConf["db/root_password"], "-h", service, "--max-allowed-packet", "256M", projectConf["db/database"])
		} else {
			cmd = exec.Command("docker", "exec", "-i", "-u", user, containerName, mysqlCommandName, "-u", "root", "-p"+projectConf["db/root_password"], "-h", service, "--max-allowed-packet", "256M", projectConf["db/database"])
		}

		progress := &progressReader{
			totalBytes: totalSize,
		}

		if ext == ".gz" {
			out, err := gzip.NewReader(selectedFile)
			if err != nil {
				logger.Fatal(err)
			}
			defer out.Close()
			progress.reader = out
		} else {
			progress.reader = selectedFile
		}

		cmd.Stdin = progress
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		done := make(chan struct{})
		go progress.printProgress(done)

		err = cmd.Run()
		close(done)
		cmdFKeys = exec.Command("docker", "exec", "-i", "-u", user, containerName, mysqlCommandName, "-u", "root", "-p"+projectConf["db/root_password"], "-h", service, "-f", "--execute", "SET FOREIGN_KEY_CHECKS=1;", projectConf["db/database"])
		if fkErr := cmdFKeys.Run(); fkErr != nil {
			logger.Fatalln("Failed to enable foreign key checks:", fkErr)
		}
		if err != nil {
			logger.Fatal(err)
		}
		fmt.Println("Database import completed successfully")
	} else {
		fmt.Println("This command is not supported for " + projectConf["platform"])
	}
}
