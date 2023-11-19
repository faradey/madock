package db

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"github.com/faradey/madock/src/cli/attr"
	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/paths"
	"github.com/jessevdk/go-flags"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type ArgsStruct struct {
	attr.Arguments
	Force         bool   `long:"force" short:"f" description:"Install Magento"`
	DBServiceName string `long:"service-name" description:"DB service name"`
	User          string `long:"user" short:"u" description:"User"`
}

func Import() {
	projectName := configs.GetProjectName()
	projectConf := configs.GetCurrentProjectConfig()

	if projectConf["PLATFORM"] != "pwa" {
		args := getArgs()

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

		globalIndex := 0
		dbsPath := paths.GetExecDirPath() + "/projects/" + projectName + "/backup/db"
		dbNames := paths.GetDBFiles(dbsPath)
		fmt.Println("Location: madock/projects/" + projectName + "/backup/db")
		if len(dbNames) == 0 {
			fmt.Println("No DB files")
		}
		for index, dbName := range dbNames {
			fmt.Println(strconv.Itoa(index+1) + ") " + filepath.Base(dbName))
			globalIndex += 1
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

		fmt.Println("Choose one of the offered variants")
		buf := bufio.NewReader(os.Stdin)
		sentence, err := buf.ReadBytes('\n')
		selected := strings.TrimSpace(string(sentence))
		selectedInt := 0
		if err != nil {
			log.Fatalln(err)
		} else {
			selectedInt, err = strconv.Atoi(selected)

			if err != nil || selectedInt > len(dbNames) {
				log.Fatal("The item you selected was not found")
			}
		}

		ext := dbNames[selectedInt-1][len(dbNames[selectedInt-1])-2:]
		out := &gzip.Reader{}

		selectedFile, err := os.Open(dbNames[selectedInt-1])
		if err != nil {
			log.Fatal(err)
		}
		defer selectedFile.Close()

		containerName := strings.ToLower(projectConf["CONTAINER_NAME_PREFIX"]) + strings.ToLower(projectName) + "-" + service + "-1"
		var cmd, cmdFKeys *exec.Cmd
		cmdFKeys = exec.Command("docker", "exec", "-i", "-u", user, containerName, "mysql", "-u", "root", "-p"+projectConf["DB_ROOT_PASSWORD"], "-h", service, "-f", "--execute", "SET FOREIGN_KEY_CHECKS=0;", projectConf["DB_DATABASE"])
		cmdFKeys.Run()
		if option != "" {
			cmd = exec.Command("docker", "exec", "-i", "-u", user, containerName, "mysql", option, "-u", "root", "-p"+projectConf["DB_ROOT_PASSWORD"], "-h", service, "--max-allowed-packet", "256M", projectConf["DB_DATABASE"])
		} else {
			cmd = exec.Command("docker", "exec", "-i", "-u", user, containerName, "mysql", "-u", "root", "-p"+projectConf["DB_ROOT_PASSWORD"], "-h", service, "--max-allowed-packet", "256M", projectConf["DB_DATABASE"])
		}

		if ext == "gz" {
			out, err = gzip.NewReader(selectedFile)
			if err != nil {
				log.Fatal(err)
			}
			cmd.Stdin = out
		} else {
			cmd.Stdin = selectedFile
		}
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		fmt.Println("Restoring database...")
		err = cmd.Run()
		cmdFKeys = exec.Command("docker", "exec", "-i", "-u", user, containerName, "mysql", "-u", "root", "-p"+projectConf["DB_ROOT_PASSWORD"], "-h", service, "-f", "--execute", "SET FOREIGN_KEY_CHECKS=1;", projectConf["DB_DATABASE"])
		cmdFKeys.Run()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Database import completed successfully")
	} else {
		fmt.Println("This command is not supported for " + projectConf["PLATFORM"])
	}
}

func getArgs() *ArgsStruct {
	args := new(ArgsStruct)
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
