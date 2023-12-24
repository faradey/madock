package db

import (
	"compress/gzip"
	"fmt"
	"github.com/alexflint/go-arg"
	"github.com/faradey/madock/src/helper/cli/attr"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/docker"
	"github.com/faradey/madock/src/helper/paths"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

type ArgsExportStruct struct {
	attr.Arguments
	Name          string   `arg:"-n,--name" help:"Name of the archive file"`
	DBServiceName string   `arg:"-s,--service" help:"DB service name. For example: db"`
	IgnoreTable   []string `arg:"--ignore-table" help:"Ignore db table"`
	User          string   `arg:"-u,--user" help:"Ignore db table"`
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

		containerName := docker.GetContainerName(projectConf, projectName, service)

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
	if attr.IsParseArgs && len(os.Args) > 2 {
		argsOrigin := os.Args[2:]
		p, err := arg.NewParser(arg.Config{
			IgnoreEnv: true,
		}, args)

		if err != nil {
			log.Fatal(err)
		}

		err = p.Parse(argsOrigin)

		if err != nil {
			log.Fatal(err)
		}
	}

	attr.IsParseArgs = false
	return args
}