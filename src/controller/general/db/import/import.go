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

	"github.com/faradey/madock/v3/src/command"
	"github.com/faradey/madock/v3/src/helper/cli/arg_struct"
	"github.com/faradey/madock/v3/src/helper/cli/attr"
	"github.com/faradey/madock/v3/src/helper/cli/fmtc"
	"github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/docker"
	"github.com/faradey/madock/v3/src/helper/logger"
	"github.com/faradey/madock/v3/src/helper/paths"
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

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"db:import"},
		Handler:  Import,
		Help:     "Import database",
		Category: "database",
		ArgsType: new(arg_struct.ControllerGeneralDbImport),
	})
}

func Import() {
	projectConf := configs.GetCurrentProjectConfig()
	args := attr.Parse(new(arg_struct.ControllerGeneralDbImport)).(*arg_struct.ControllerGeneralDbImport)

	service := "db"
	if args.DBServiceName != "" {
		service = args.DBServiceName
	}

	projectName := configs.GetProjectName()

	filePath := resolveImportFilePath(args, projectName)
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

	dbType := configs.GetDbType(projectConf)

	switch dbType {
	case "postgresql":
		importPostgresql(containerName, projectConf, args, service, selectedFile, ext, totalSize)
	case "mongodb":
		importMongodb(containerName, projectConf, args, selectedFile, totalSize)
	default:
		importMysql(containerName, projectConf, args, service, selectedFile, ext, totalSize)
	}

	fmt.Println("Database import completed successfully")
}

func resolveImportFilePath(args *arg_struct.ControllerGeneralDbImport, projectName string) string {
	if args.File != "" {
		var filePath string
		if !filepath.IsAbs(args.File) {
			filePath = filepath.Join(paths.GetRunDirPath(), args.File)
		} else {
			filePath = args.File
		}
		if !paths.IsFileExist(filePath) {
			logger.Fatal("File not found: " + filePath)
		}
		return filePath
	}

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

	fmt.Println("")
	fmtc.TitleLn("Select database file to import:")
	selector := fmtc.NewInteractiveSelector("Database File", displayNames, 0)
	selectedIdx, _ := selector.Run()

	return dbNames[selectedIdx]
}

func importMysql(containerName string, projectConf map[string]string, args *arg_struct.ControllerGeneralDbImport, service string, selectedFile *os.File, ext string, totalSize int64) {
	option := ""
	if args.Force {
		option = "-f"
	}

	user := "mysql"
	if args.User != "" {
		user = args.User
	}

	mysqlCommandName := "mysql"
	if projectConf["db/repository"] == "mariadb" && configs.CompareVersions(projectConf["db/version"], "10.5") != -1 {
		mysqlCommandName = "mariadb"
	}

	fkCmd, fkErr := docker.PrepareContainerExec(containerName, user, false, mysqlCommandName, "-u", "root", "-p"+projectConf["db/root_password"], "-h", service, "-f", "--execute", "SET FOREIGN_KEY_CHECKS=0;", projectConf["db/database"])
	if fkErr != nil {
		logger.Fatalln("Failed to prepare foreign key command:", fkErr)
	}
	if err := fkCmd.Run(); err != nil {
		logger.Fatalln("Failed to disable foreign key checks:", err)
	}
	var cmd *exec.Cmd
	if option != "" {
		cmd, _ = docker.PrepareContainerExec(containerName, user, false, mysqlCommandName, option, "-u", "root", "-p"+projectConf["db/root_password"], "-h", service, "--max-allowed-packet", "256M", projectConf["db/database"])
	} else {
		cmd, _ = docker.PrepareContainerExec(containerName, user, false, mysqlCommandName, "-u", "root", "-p"+projectConf["db/root_password"], "-h", service, "--max-allowed-packet", "256M", projectConf["db/database"])
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

	err := cmd.Run()
	close(done)
	fkCmd2, _ := docker.PrepareContainerExec(containerName, user, false, mysqlCommandName, "-u", "root", "-p"+projectConf["db/root_password"], "-h", service, "-f", "--execute", "SET FOREIGN_KEY_CHECKS=1;", projectConf["db/database"])
	if fkErr := fkCmd2.Run(); fkErr != nil {
		logger.Fatalln("Failed to enable foreign key checks:", fkErr)
	}
	if err != nil {
		logger.Fatal(err)
	}
}

func importPostgresql(containerName string, projectConf map[string]string, args *arg_struct.ControllerGeneralDbImport, service string, selectedFile *os.File, ext string, totalSize int64) {
	user := "postgres"
	if args.User != "" {
		user = args.User
	}

	cmd, prepErr := docker.PrepareContainerExec(containerName, user, false, "psql", "-U", projectConf["db/user"], "-h", service, projectConf["db/database"])
	if prepErr != nil {
		logger.Fatal(prepErr)
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

	err := cmd.Run()
	close(done)
	if err != nil {
		logger.Fatal(err)
	}
}

func importMongodb(containerName string, projectConf map[string]string, args *arg_struct.ControllerGeneralDbImport, selectedFile *os.File, totalSize int64) {
	user := "root"
	if args.User != "" {
		user = args.User
	}

	cmd, prepErr := docker.PrepareContainerExec(containerName, user, false, "bash", "-c", "mongorestore --username="+projectConf["db/user"]+" --password="+projectConf["db/password"]+" --authenticationDatabase=admin --db="+projectConf["db/database"]+" --archive --gzip --drop")
	if prepErr != nil {
		logger.Fatal(prepErr)
	}

	progress := &progressReader{
		totalBytes: totalSize,
	}
	progress.reader = selectedFile

	cmd.Stdin = progress
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	done := make(chan struct{})
	go progress.printProgress(done)

	err := cmd.Run()
	close(done)
	if err != nil {
		logger.Fatal(err)
	}
}
