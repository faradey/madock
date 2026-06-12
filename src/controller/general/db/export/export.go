package export

import (
	"compress/gzip"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/faradey/madock/v3/src/command"
	"github.com/faradey/madock/v3/src/helper/cli/arg_struct"
	"github.com/faradey/madock/v3/src/helper/cli/attr"
	"github.com/faradey/madock/v3/src/helper/cli/output"
	"github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/docker"
	"github.com/faradey/madock/v3/src/helper/logger"
	"github.com/faradey/madock/v3/src/helper/paths"
)

type DbExportOutput struct {
	File string `json:"file"`
}

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"db:export"},
		Handler:  Export,
		Help:     "Export database",
		Category: "database",
		ArgsType: new(arg_struct.ControllerGeneralDbExport),
	})
}

func Export() {
	projectConf := configs.GetCurrentProjectConfig()
	args := attr.Parse(new(arg_struct.ControllerGeneralDbExport)).(*arg_struct.ControllerGeneralDbExport)

	name := strings.TrimSpace(args.Name)
	if len(name) > 0 {
		name += "_"
	}

	service := "db"
	if args.DBServiceName != "" {
		service = args.DBServiceName
	}

	projectName := configs.GetProjectName()
	containerName := docker.GetContainerName(projectConf, projectName, service)
	dbsPath := paths.MakeDirsByPath(paths.GetExecDirPath() + "/projects/" + projectName + "/backup/db/")

	dbType := configs.GetDbType(projectConf)

	var filePath string
	switch dbType {
	case "postgresql":
		filePath = exportPostgresql(containerName, projectConf, args, name, service, dbsPath)
	case "mongodb":
		filePath = exportMongodb(containerName, projectConf, args, name, dbsPath)
	default:
		filePath = exportMysql(containerName, projectConf, args, name, service, dbsPath)
	}

	if args.Json {
		if err := output.PrintJSON(DbExportOutput{File: filePath}); err != nil {
			logger.Fatal(err)
		}
		return
	}

	fmt.Println("Database export completed successfully")
	fmt.Println(filePath)
}

func exportMysql(containerName string, projectConf map[string]string, args *arg_struct.ControllerGeneralDbExport, name, service, dbsPath string) string {
	ignoreTablesStr := ""
	ignoreTables := args.IgnoreTable
	if len(ignoreTables) > 0 {
		ignoreTablesStr = " --ignore-table=" + projectConf["db/database"] + "." + strings.Join(ignoreTables, " --ignore-table="+projectConf["db/database"]+".")
	}

	user := "mysql"
	if args.User != "" {
		user = args.User
	}

	filePath := dbsPath + "local_" + name + time.Now().Format("2006-01-02_15-04-05") + ".sql.gz"
	selectedFile, err := os.Create(filePath)
	if err != nil {
		logger.Fatal(err)
	}
	defer selectedFile.Close()
	writer := gzip.NewWriter(selectedFile)
	defer writer.Close()

	mysqldumpCommandName := "mysqldump"
	if projectConf["db/repository"] == "mariadb" && configs.CompareVersions(projectConf["db/version"], "10.5") != -1 {
		mysqldumpCommandName = "mariadb-dump"
	}

	// set -o pipefail so a mysqldump/mariadb-dump failure (e.g. unknown database,
	// auth error) propagates instead of being masked by the trailing `| sed`,
	// which would otherwise report success while writing an empty dump.
	cmd, prepErr := docker.PrepareContainerExec(containerName, user, false, "bash", "-c", "set -o pipefail; "+mysqldumpCommandName+" -u root -p"+projectConf["db/root_password"]+" -v -h "+service+ignoreTablesStr+" "+projectConf["db/database"]+" | sed -e 's/DEFINER[ ]*=[ ]*[^*]*\\*/\\*/'")
	if prepErr != nil {
		logger.Fatal(prepErr)
	}
	cmd.Stdout = writer
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	docker.NotifyExecDone(containerName, []string{"bash", "-c", "mysqldump..."}, err)
	if err != nil {
		writer.Close()
		selectedFile.Close()
		_ = os.Remove(filePath)
		logger.Fatal(err)
	}

	return filePath
}

func exportPostgresql(containerName string, projectConf map[string]string, args *arg_struct.ControllerGeneralDbExport, name, service, dbsPath string) string {
	user := "postgres"
	if args.User != "" {
		user = args.User
	}

	filePath := dbsPath + "local_" + name + time.Now().Format("2006-01-02_15-04-05") + ".sql.gz"
	selectedFile, err := os.Create(filePath)
	if err != nil {
		logger.Fatal(err)
	}
	defer selectedFile.Close()
	writer := gzip.NewWriter(selectedFile)
	defer writer.Close()

	ignoreTablesStr := ""
	ignoreTables := args.IgnoreTable
	if len(ignoreTables) > 0 {
		for _, t := range ignoreTables {
			ignoreTablesStr += " --exclude-table=" + t
		}
	}

	cmd, prepErr := docker.PrepareContainerExec(containerName, user, false, "bash", "-c", "PGPASSWORD='"+projectConf["db/password"]+"' pg_dump -U "+projectConf["db/user"]+" -h "+service+ignoreTablesStr+" "+projectConf["db/database"])
	if prepErr != nil {
		logger.Fatal(prepErr)
	}
	cmd.Stdout = writer
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	docker.NotifyExecDone(containerName, []string{"bash", "-c", "pg_dump..."}, err)
	if err != nil {
		writer.Close()
		selectedFile.Close()
		_ = os.Remove(filePath)
		logger.Fatal(err)
	}

	return filePath
}

func exportMongodb(containerName string, projectConf map[string]string, args *arg_struct.ControllerGeneralDbExport, name, dbsPath string) string {
	user := "root"
	if args.User != "" {
		user = args.User
	}

	filePath := dbsPath + "local_" + name + time.Now().Format("2006-01-02_15-04-05") + ".archive.gz"
	selectedFile, err := os.Create(filePath)
	if err != nil {
		logger.Fatal(err)
	}
	defer selectedFile.Close()

	cmd, prepErr := docker.PrepareContainerExec(containerName, user, false, "bash", "-c", "mongodump --username="+projectConf["db/user"]+" --password="+projectConf["db/password"]+" --authenticationDatabase=admin --db="+projectConf["db/database"]+" --archive --gzip")
	if prepErr != nil {
		logger.Fatal(prepErr)
	}
	cmd.Stdout = selectedFile
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	docker.NotifyExecDone(containerName, []string{"bash", "-c", "mongodump..."}, err)
	if err != nil {
		selectedFile.Close()
		_ = os.Remove(filePath)
		logger.Fatal(err)
	}

	return filePath
}
