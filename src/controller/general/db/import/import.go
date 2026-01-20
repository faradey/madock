package _import

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/faradey/madock/src/helper/cli/arg_struct"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/cli/fmtc"
	"github.com/faradey/madock/src/helper/configs"
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

		var filePath string
		projectName := configs.GetProjectName()

		if args.File != "" {
			// Use file path from argument
			if !filepath.IsAbs(args.File) {
				filePath = filepath.Join(paths.GetRunDirPath(), args.File)
			} else {
				filePath = args.File
			}
			if !paths.IsFileExist(filePath) {
				logger.Fatal("File not found: " + filePath)
			}
		} else {
			// Interactive file selection
			dbsPath := paths.GetExecDirPath() + "/projects/" + projectName + "/backup/db"
			var dbNames []string
			var displayNames []string

			if paths.IsFileExist(dbsPath) {
				backupFiles := paths.GetDBFiles(dbsPath)
				if len(backupFiles) > 0 {
					for _, dbName := range backupFiles {
						dbNames = append(dbNames, dbName)
						displayNames = append(displayNames, filepath.Base(dbName)+" (backup)")
					}
				}
			}

			dbsPath = paths.GetRunDirPath()
			projectFiles := paths.GetDBFiles(dbsPath)
			if len(projectFiles) > 0 {
				for _, dbName := range projectFiles {
					dbNames = append(dbNames, dbName)
					displayNames = append(displayNames, filepath.Base(dbName)+" (project)")
				}
			}

			if len(dbNames) == 0 {
				logger.Fatal("No database files found for import")
			}

			// Use interactive selector for file selection
			fmt.Println("")
			fmtc.TitleLn("Select database file to import:")
			selector := fmtc.NewInteractiveSelector("Database File", displayNames, 0)
			selectedIdx, _ := selector.Run()

			filePath = dbNames[selectedIdx]
		}
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
		if projectConf["db/repository"] == "mariadb" && configs.CompareVersions(projectConf["db/version"], "10.5") != -1 {
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
