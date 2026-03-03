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
	"github.com/faradey/madock/v3/src/helper/configs"
	"github.com/faradey/madock/v3/src/helper/docker"
	"github.com/faradey/madock/v3/src/helper/logger"
	"github.com/faradey/madock/v3/src/helper/paths"
)

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

	switch dbType {
	case "postgresql":
		exportPostgresql(containerName, projectConf, args, name, service, dbsPath)
	case "mongodb":
		exportMongodb(containerName, projectConf, args, name, dbsPath)
	default:
		exportMysql(containerName, projectConf, args, name, service, dbsPath)
	}

	fmt.Println("Database export completed successfully")
}

func exportMysql(containerName string, projectConf map[string]string, args *arg_struct.ControllerGeneralDbExport, name, service, dbsPath string) {
	ignoreTablesStr := ""
	ignoreTables := args.IgnoreTable
	if len(ignoreTables) > 0 {
		ignoreTablesStr = " --ignore-table=" + projectConf["db/database"] + "." + strings.Join(ignoreTables, " --ignore-table="+projectConf["db/database"]+".")
	}

	user := "mysql"
	if args.User != "" {
		user = args.User
	}

	selectedFile, err := os.Create(dbsPath + "local_" + name + time.Now().Format("2006-01-02_15-04-05") + ".sql.gz")
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

	cmd, prepErr := docker.PrepareContainerExec(containerName, user, false, "bash", "-c", mysqldumpCommandName+" -u root -p"+projectConf["db/root_password"]+" -v -h "+service+ignoreTablesStr+" "+projectConf["db/database"]+" | sed -e 's/DEFINER[ ]*=[ ]*[^*]*\\*/\\*/'")
	if prepErr != nil {
		logger.Fatal(prepErr)
	}
	cmd.Stdout = writer
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	docker.NotifyExecDone(containerName, []string{"bash", "-c", "mysqldump..."}, err)
	if err != nil {
		logger.Fatal(err)
	}
}

func exportPostgresql(containerName string, projectConf map[string]string, args *arg_struct.ControllerGeneralDbExport, name, service, dbsPath string) {
	user := "postgres"
	if args.User != "" {
		user = args.User
	}

	selectedFile, err := os.Create(dbsPath + "local_" + name + time.Now().Format("2006-01-02_15-04-05") + ".sql.gz")
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

	cmd, prepErr := docker.PrepareContainerExec(containerName, user, false, "bash", "-c", "pg_dump -U "+projectConf["db/user"]+" -h "+service+ignoreTablesStr+" "+projectConf["db/database"])
	if prepErr != nil {
		logger.Fatal(prepErr)
	}
	cmd.Stdout = writer
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	docker.NotifyExecDone(containerName, []string{"bash", "-c", "pg_dump..."}, err)
	if err != nil {
		logger.Fatal(err)
	}
}

func exportMongodb(containerName string, projectConf map[string]string, args *arg_struct.ControllerGeneralDbExport, name, dbsPath string) {
	user := "root"
	if args.User != "" {
		user = args.User
	}

	selectedFile, err := os.Create(dbsPath + "local_" + name + time.Now().Format("2006-01-02_15-04-05") + ".archive.gz")
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
		logger.Fatal(err)
	}
}
