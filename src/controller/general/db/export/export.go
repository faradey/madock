package export

import (
	"compress/gzip"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/faradey/madock/src/command"
	"github.com/faradey/madock/src/helper/cli/arg_struct"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/docker"
	"github.com/faradey/madock/src/helper/logger"
	"github.com/faradey/madock/src/helper/paths"
)

func init() {
	command.Register(&command.Definition{
		Aliases:  []string{"db:export"},
		Handler:  Export,
		Help:     "Export database",
		Category: "database",
	})
}

func Export() {
	projectConf := configs.GetCurrentProjectConfig()
	args := attr.Parse(new(arg_struct.ControllerGeneralDbExport)).(*arg_struct.ControllerGeneralDbExport)

	name := strings.TrimSpace(args.Name)
	if len(name) > 0 {
		name += "_"
	}

	ignoreTablesStr := ""
	ignoreTables := args.IgnoreTable
	if len(ignoreTables) > 0 {
		ignoreTablesStr = " --ignore-table=" + projectConf["db/database"] + "." + strings.Join(ignoreTables, " --ignore-table="+projectConf["db/database"]+".")
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
	containerName := docker.GetContainerName(projectConf, projectName, service)

	dbsPath := paths.MakeDirsByPath(paths.GetExecDirPath() + "/projects/" + projectName + "/backup/db/")
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

	cmd := exec.Command("docker", "exec", "-i", "-u", user, containerName, "bash", "-c", mysqldumpCommandName+" -u root -p"+projectConf["db/root_password"]+" -v -h "+service+ignoreTablesStr+" "+projectConf["db/database"]+" | sed -e 's/DEFINER[ ]*=[ ]*[^*]*\\*/\\*/'")
	cmd.Stdout = writer
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		logger.Fatal(err)
	}
	fmt.Println("Database export completed successfully")
}
