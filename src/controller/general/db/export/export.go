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
	"github.com/faradey/madock/v3/src/helper/dbtarget"
	"github.com/faradey/madock/v3/src/helper/docker"
	"github.com/faradey/madock/v3/src/helper/logger"
	"github.com/faradey/madock/v3/src/helper/paths"
)

type DbExportOutput struct {
	File string `json:"file"`
}

// dumpPrefix names the source in the file name. A dump taken from a shared
// database sits in this project's backup directory but holds another project's
// schema — and everything every other consumer of that schema keeps in it.
// Calling that file "local_" is how it ends up restored, or uploaded, as if it
// were this project's own data.
func dumpPrefix(target dbtarget.Target) string {
	if target.Shared {
		return "shared-" + target.Project + "_"
	}
	return "local_"
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
	target := dbtarget.MustResolve(projectConf, projectName, service)
	// The dump belongs to the project that asked for it, even when the data
	// lives on another project's server.
	dbsPath := paths.MakeDirsByPath(paths.GetExecDirPath() + "/projects/" + projectName + "/backup/db/")

	var filePath string
	switch target.Type {
	case "postgresql":
		filePath = exportPostgresql(target, args, name, dbsPath)
	case "mongodb":
		filePath = exportMongodb(target, args, name, dbsPath)
	default:
		filePath = exportMysql(target, args, name, dbsPath)
	}

	if args.Json {
		if err := output.PrintJSON(DbExportOutput{File: filePath}); err != nil {
			logger.Fatal(err)
		}
		return
	}

	fmt.Println("Database export completed successfully")
	if target.Shared {
		fmt.Println("Source: " + target.Origin())
	}
	fmt.Println(filePath)
}

func exportMysql(target dbtarget.Target, args *arg_struct.ControllerGeneralDbExport, name, dbsPath string) string {
	ignoreTablesStr := ""
	ignoreTables := args.IgnoreTable
	if len(ignoreTables) > 0 {
		ignoreTablesStr = " --ignore-table=" + target.Database + "." + strings.Join(ignoreTables, " --ignore-table="+target.Database+".")
	}

	user := "mysql"
	if args.User != "" {
		user = args.User
	}

	filePath := dbsPath + dumpPrefix(target) + name + time.Now().Format("2006-01-02_15-04-05") + ".sql.gz"
	selectedFile, err := os.Create(filePath)
	if err != nil {
		logger.Fatal(err)
	}
	defer selectedFile.Close()
	writer := gzip.NewWriter(selectedFile)
	defer writer.Close()

	// set -o pipefail so a mysqldump/mariadb-dump failure (e.g. unknown database,
	// auth error) propagates instead of being masked by the trailing `| sed`,
	// which would otherwise report success while writing an empty dump.
	cmd, prepErr := docker.PrepareContainerExec(target.Container, user, false, "bash", "-c", "set -o pipefail; "+target.MySQLDump()+" -u root -p"+target.RootPassword+" -v -h "+target.Host+ignoreTablesStr+" "+target.Database+" | sed -e 's/DEFINER[ ]*=[ ]*[^*]*\\*/\\*/'")
	if prepErr != nil {
		logger.Fatal(prepErr)
	}
	cmd.Stdout = writer
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	docker.NotifyExecDone(target.Container, []string{"bash", "-c", "mysqldump..."}, err)
	if err != nil {
		writer.Close()
		selectedFile.Close()
		_ = os.Remove(filePath)
		logger.Fatal(err)
	}

	return filePath
}

func exportPostgresql(target dbtarget.Target, args *arg_struct.ControllerGeneralDbExport, name, dbsPath string) string {
	user := "postgres"
	if args.User != "" {
		user = args.User
	}

	filePath := dbsPath + dumpPrefix(target) + name + time.Now().Format("2006-01-02_15-04-05") + ".sql.gz"
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

	cmd, prepErr := docker.PrepareContainerExec(target.Container, user, false, "bash", "-c", "PGPASSWORD='"+target.Password+"' pg_dump -U "+target.User+" -h "+target.Host+ignoreTablesStr+" "+target.Database)
	if prepErr != nil {
		logger.Fatal(prepErr)
	}
	cmd.Stdout = writer
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	docker.NotifyExecDone(target.Container, []string{"bash", "-c", "pg_dump..."}, err)
	if err != nil {
		writer.Close()
		selectedFile.Close()
		_ = os.Remove(filePath)
		logger.Fatal(err)
	}

	return filePath
}

func exportMongodb(target dbtarget.Target, args *arg_struct.ControllerGeneralDbExport, name, dbsPath string) string {
	user := "root"
	if args.User != "" {
		user = args.User
	}

	filePath := dbsPath + dumpPrefix(target) + name + time.Now().Format("2006-01-02_15-04-05") + ".archive.gz"
	selectedFile, err := os.Create(filePath)
	if err != nil {
		logger.Fatal(err)
	}
	defer selectedFile.Close()

	cmd, prepErr := docker.PrepareContainerExec(target.Container, user, false, "bash", "-c", "mongodump --username="+target.User+" --password="+target.Password+" --authenticationDatabase=admin --db="+target.Database+" --archive --gzip")
	if prepErr != nil {
		logger.Fatal(prepErr)
	}
	cmd.Stdout = selectedFile
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	docker.NotifyExecDone(target.Container, []string{"bash", "-c", "mongodump..."}, err)
	if err != nil {
		selectedFile.Close()
		_ = os.Remove(filePath)
		logger.Fatal(err)
	}

	return filePath
}
