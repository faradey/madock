package builder

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/faradey/madock/src/cli/attr"
	"github.com/faradey/madock/src/cli/fmtc"
	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/paths"
)

func DbImport() {
	option := ""
	if attr.Options.Force {
		option = "-f"
	}
	dbServiceName := "db"
	if attr.Options.DBServiceName != "" {
		dbServiceName = attr.Options.DBServiceName
	}
	projectName := configs.GetProjectName()
	projectConfig := configs.GetCurrentProjectConfig()

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

	containerName := strings.ToLower(projectConfig["CONTAINER_NAME_PREFIX"]) + strings.ToLower(projectName) + "-" + dbServiceName + "-1"
	var cmd, cmdFKeys *exec.Cmd
	cmdFKeys = exec.Command("docker", "exec", "-i", "-u", "mysql", containerName, "mysql", "-u", "root", "-p"+projectConfig["DB_ROOT_PASSWORD"], "-h", dbServiceName, "-f", "--execute", "SET FOREIGN_KEY_CHECKS=0;", projectConfig["DB_DATABASE"])
	cmdFKeys.Run()
	if option != "" {
		cmd = exec.Command("docker", "exec", "-i", "-u", "mysql", containerName, "mysql", option, "-u", "root", "-p"+projectConfig["DB_ROOT_PASSWORD"], "-h", dbServiceName, "--max-allowed-packet", "256M", projectConfig["DB_DATABASE"])
	} else {
		cmd = exec.Command("docker", "exec", "-i", "-u", "mysql", containerName, "mysql", "-u", "root", "-p"+projectConfig["DB_ROOT_PASSWORD"], "-h", dbServiceName, "--max-allowed-packet", "256M", projectConfig["DB_DATABASE"])
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
	cmdFKeys = exec.Command("docker", "exec", "-i", "-u", "mysql", containerName, "mysql", "-u", "root", "-p"+projectConfig["DB_ROOT_PASSWORD"], "-h", dbServiceName, "-f", "--execute", "SET FOREIGN_KEY_CHECKS=1;", projectConfig["DB_DATABASE"])
	cmdFKeys.Run()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Database import completed successfully")
}

func DbExport() {
	projectName := configs.GetProjectName()
	projectConfig := configs.GetCurrentProjectConfig()
	name := strings.TrimSpace(attr.Options.Name)
	if len(name) > 0 {
		name += "_"
	}

	ignoreTablesStr := ""
	ignoreTables := attr.Options.IgnoreTable
	if len(ignoreTables) > 0 {
		ignoreTablesStr = " --ignore-table=" + projectConfig["DB_DATABASE"] + "." + strings.Join(ignoreTables, " --ignore-table="+projectConfig["DB_DATABASE"]+".")
	}

	dbServiceName := "db"
	if attr.Options.DBServiceName != "" {
		dbServiceName = attr.Options.DBServiceName
	}

	containerName := strings.ToLower(projectConfig["CONTAINER_NAME_PREFIX"]) + strings.ToLower(projectName) + "-" + dbServiceName + "-1"

	dbsPath := paths.GetExecDirPath() + "/projects/" + projectName + "/backup/db/"
	selectedFile, err := os.Create(dbsPath + "local_" + name + time.Now().Format("2006-01-02_15-04-05") + ".sql.gz")
	if err != nil {
		log.Fatal(err)
	}
	defer selectedFile.Close()
	writer := gzip.NewWriter(selectedFile)
	defer writer.Close()
	cmd := exec.Command("docker", "exec", "-i", "-u", "mysql", containerName, "bash", "-c", "mysqldump -u root -p"+projectConfig["DB_ROOT_PASSWORD"]+" -v -h "+dbServiceName+ignoreTablesStr+" "+projectConfig["DB_DATABASE"]+" | sed -e 's/DEFINER[ ]*=[ ]*[^*]*\\*/\\*/'")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdout = writer
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Database export completed successfully")
}

func DbInfo() {
	projectConfig := configs.GetCurrentProjectConfig()
	portsFile := paths.GetExecDirPath() + "/aruntime/ports.conf"
	portsConfig := configs.ParseFile(portsFile)
	port, err := strconv.Atoi(portsConfig[configs.GetProjectName()])
	if err != nil {
		log.Fatal(err)
	}
	fmtc.SuccessLn("host: db")
	fmtc.SuccessLn("name: " + projectConfig["DB_DATABASE"])
	fmtc.SuccessLn("user: " + projectConfig["DB_USER"])
	fmtc.SuccessLn("password: " + projectConfig["DB_PASSWORD"])
	fmtc.SuccessLn("root password: " + projectConfig["DB_ROOT_PASSWORD"])
	fmtc.SuccessLn("remote HOST:PORT: " + "localhost:" + strconv.Itoa(17000+((port-1)*20)+4))
}
