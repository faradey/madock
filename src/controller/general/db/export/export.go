package export

import (
	"compress/gzip"
	"fmt"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/docker"
	"github.com/faradey/madock/src/helper/logger"
	"github.com/faradey/madock/src/helper/paths"
	"os"
	"os/exec"
	"strings"
	"time"
)

type ArgsStruct struct {
	attr.Arguments
	Name          string   `arg:"-n,--name" help:"Name of the archive file"`
	DBServiceName string   `arg:"-s,--service" help:"DB service name. For example: db"`
	IgnoreTable   []string `arg:"--ignore-table" help:"Ignore db table"`
	User          string   `arg:"-u,--user" help:"Ignore db table"`
}

func Export() {
	projectName := configs.GetProjectName()
	projectConf := configs.GetCurrentProjectConfig()
	if projectConf["platform"] != "pwa" {
		args := attr.Parse(new(ArgsStruct)).(*ArgsStruct)

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

		containerName := docker.GetContainerName(projectConf, projectName, service)

		dbsPath := paths.MakeDirsByPath(paths.GetExecDirPath() + "/projects/" + projectName + "/backup/db/")
		selectedFile, err := os.Create(dbsPath + "local_" + name + time.Now().Format("2006-01-02_15-04-05") + ".sql.gz")
		if err != nil {
			logger.Fatal(err)
		}
		defer selectedFile.Close()
		writer := gzip.NewWriter(selectedFile)
		defer writer.Close()
		cmd := exec.Command("docker", "exec", "-i", "-u", user, containerName, "bash", "-c", "mysqldump -u root -p"+projectConf["db/root_password"]+" -v -h "+service+ignoreTablesStr+" "+projectConf["db/database"]+" | sed -e 's/DEFINER[ ]*=[ ]*[^*]*\\*/\\*/'")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdout = writer
		err = cmd.Run()
		if err != nil {
			logger.Fatal(err)
		}
		fmt.Println("Database export completed successfully")
	} else {
		fmt.Println("This command is not supported for " + projectConf["platform"])
	}
}
