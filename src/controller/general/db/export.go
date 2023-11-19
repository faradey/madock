package db

import (
	"compress/gzip"
	"fmt"
	"github.com/faradey/madock/src/cli/attr"
	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/paths"
	"github.com/jessevdk/go-flags"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

type ArgsExportStruct struct {
	attr.Arguments
	Name          string   `long:"name" short:"n" description:"Name of the archive file"`
	DBServiceName string   `long:"service" short:"s" description:"DB service name. For example: db"`
	IgnoreTable   []string `long:"ignore-table" description:"Ignore db table"`
	User          string   `long:"user" short:"u" description:"User"`
}

func Export() {

	projectName := configs.GetProjectName()
	projectConf := configs.GetCurrentProjectConfig()
	if projectConf["PLATFORM"] != "pwa" {
		args := getExportArgs()

		name := strings.TrimSpace(args.Name)
		if len(name) > 0 {
			name += "_"
		}

		ignoreTablesStr := ""
		ignoreTables := args.IgnoreTable
		if len(ignoreTables) > 0 {
			ignoreTablesStr = " --ignore-table=" + projectConf["DB_DATABASE"] + "." + strings.Join(ignoreTables, " --ignore-table="+projectConf["DB_DATABASE"]+".")
		}

		service := "db"
		if args.DBServiceName != "" {
			service = args.DBServiceName
		}

		user := "mysql"
		if args.User != "" {
			user = args.User
		}

		containerName := strings.ToLower(projectConf["CONTAINER_NAME_PREFIX"]) + strings.ToLower(projectName) + "-" + service + "-1"

		dbsPath := paths.GetExecDirPath() + "/projects/" + projectName + "/backup/db/"
		selectedFile, err := os.Create(dbsPath + "local_" + name + time.Now().Format("2006-01-02_15-04-05") + ".sql.gz")
		if err != nil {
			log.Fatal(err)
		}
		defer selectedFile.Close()
		writer := gzip.NewWriter(selectedFile)
		defer writer.Close()
		cmd := exec.Command("docker", "exec", "-i", "-u", user, containerName, "bash", "-c", "mysqldump -u root -p"+projectConf["DB_ROOT_PASSWORD"]+" -v -h "+service+ignoreTablesStr+" "+projectConf["DB_DATABASE"]+" | sed -e 's/DEFINER[ ]*=[ ]*[^*]*\\*/\\*/'")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdout = writer
		err = cmd.Run()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Database export completed successfully")
	} else {
		fmt.Println("This command is not supported for " + projectConf["PLATFORM"])
	}
}

func getExportArgs() *ArgsExportStruct {
	args := new(ArgsExportStruct)
	if len(os.Args) > 2 {
		argsOrigin := os.Args[2:]
		var err error
		_, err = flags.ParseArgs(args, argsOrigin)

		if err != nil {
			log.Fatal(err)
		}
	}

	return args
}
