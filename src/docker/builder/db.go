package builder

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/paths"
)

func DbImport(option string) {
	if len(option) > 0 && option != "-f" {
		option = ""
	}
	projectName := paths.GetRunDirName()
	projectConfig := configs.GetCurrentProjectConfig()
	dbsPath := paths.GetExecDirPath() + "/projects/" + projectName + "/backup/db"
	dbNames := paths.GetDBFiles(dbsPath)
	for index, dbName := range dbNames {
		fmt.Println(strconv.Itoa(index+1) + ") " + dbName)
	}
	fmt.Println("Choose one of the options offered")
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

	selectedFile, err := os.Open(dbsPath + "/" + dbNames[selectedInt-1])
	if err != nil {
		log.Fatal(err)
	}
	defer selectedFile.Close()

	var cmd, cmdFKeys *exec.Cmd
	cmdFKeys = exec.Command("docker", "exec", "-i", "-u", "mysql", strings.ToLower(projectName)+"-db-1", "mysql", option, "-u", "root", "-p"+projectConfig["DB_ROOT_PASSWORD"], "-h", "db", "-f", "--execute", "SET FOREIGN_KEY_CHECKS=0;", projectConfig["DB_DATABASE"])
	cmdFKeys.Run()
	if option != "" {
		cmd = exec.Command("docker", "exec", "-i", "-u", "mysql", strings.ToLower(projectName)+"-db-1", "mysql", option, "-u", "root", "-p"+projectConfig["DB_ROOT_PASSWORD"], "-h", "db", "--max-allowed-packet", "256M", projectConfig["DB_DATABASE"])
	} else {
		cmd = exec.Command("docker", "exec", "-i", "-u", "mysql", strings.ToLower(projectName)+"-db-1", "mysql", "-u", "root", "-p"+projectConfig["DB_ROOT_PASSWORD"], "-h", "db", "--max-allowed-packet", "256M", projectConfig["DB_DATABASE"])
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
	err = cmd.Run()
	cmdFKeys = exec.Command("docker", "exec", "-i", "-u", "mysql", strings.ToLower(projectName)+"-db-1", "mysql", option, "-u", "root", "-p"+projectConfig["DB_ROOT_PASSWORD"], "-h", "db", "-f", "--execute", "SET FOREIGN_KEY_CHECKS=1;", projectConfig["DB_DATABASE"])
	cmdFKeys.Run()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Database import completed successfully")
}

func DbExport() {
	projectName := paths.GetRunDirName()
	projectConfig := configs.GetCurrentProjectConfig()
	dbsPath := paths.GetExecDirPath() + "/projects/" + projectName + "/backup/db/"
	selectedFile, err := os.Create(dbsPath + "local-" + time.Now().Format("2006-01-02_15-04-05") + ".sql.gz")
	if err != nil {
		log.Fatal(err)
	}
	defer selectedFile.Close()
	writer := gzip.NewWriter(selectedFile)
	defer writer.Close()
	cmd := exec.Command("docker", "exec", "-i", "-u", "mysql", strings.ToLower(projectName)+"-db-1", "mysqldump", "-u", "root", "-p"+projectConfig["DB_ROOT_PASSWORD"], "-v", "-h", "db", projectConfig["DB_DATABASE"])
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdout = writer
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Database export completed successfully")
}

func DbSoftClean() {
	fmt.Println("Start cleaning up the database")
	projectName := paths.GetRunDirName()
	projectConfig := configs.GetCurrentProjectConfig()
	tablesList := "TRUNCATE TABLE dataflow_batch_export;"
	tablesList += "TRUNCATE TABLE dataflow_batch_import;"
	tablesList += "TRUNCATE TABLE log_customer;"
	tablesList += "TRUNCATE TABLE log_quote;"
	tablesList += "TRUNCATE TABLE log_summary;"
	tablesList += "TRUNCATE TABLE log_summary_type;"
	tablesList += "TRUNCATE TABLE log_url;"
	tablesList += "TRUNCATE TABLE log_url_info;"
	tablesList += "TRUNCATE TABLE log_visitor;"
	tablesList += "TRUNCATE TABLE log_visitor_info;"
	tablesList += "TRUNCATE TABLE log_visitor_online;"
	tablesList += "TRUNCATE TABLE report_viewed_product_index;"
	tablesList += "TRUNCATE TABLE report_compared_product_index;"
	tablesList += "TRUNCATE TABLE report_event;"
	tablesList += "TRUNCATE TABLE index_event;"
	tablesList += "TRUNCATE TABLE catalog_compare_item;"
	tablesList += "TRUNCATE TABLE catalogindex_aggregation;"
	tablesList += "TRUNCATE TABLE catalogindex_aggregation_tag;"
	tablesList += "TRUNCATE TABLE catalogindex_aggregation_to_tag;"
	tablesList += "TRUNCATE TABLE adminnotification_inbox;"
	tablesList += "TRUNCATE TABLE aw_core_logger;"
	tablesList += "TRUNCATE TABLE kiwicommerce_activity_log;"
	tablesList += "TRUNCATE TABLE kiwicommerce_activity_detail;"
	tablesList += "TRUNCATE TABLE kiwicommerce_activity;"
	tablesList += "TRUNCATE TABLE kiwicommerce_login_activity;"
	tablesList += "TRUNCATE TABLE amasty_amsmtp_log;"
	tablesList += "TRUNCATE TABLE search_query;"
	tablesList += "TRUNCATE TABLE persistent_session;"
	tablesList += "TRUNCATE TABLE mailchimp_errors;"
	tablesList += "TRUNCATE TABLE session;"

	var b strings.Builder
	cmdTemp := exec.Command("docker", "exec", "-i", "-u", "mysql", strings.ToLower(projectName)+"-db-1", "mysql", "-u", "root", "-p"+projectConfig["DB_ROOT_PASSWORD"], "-h", "db", "-f", "--execute", "SELECT concat('TRUNCATE TABLE `', TABLE_NAME, '`;') FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_NAME LIKE 'catalogrule_product%__temp%' OR TABLE_NAME LIKE 'quote%'", projectConfig["DB_DATABASE"])
	cmdTemp.Stdout = &b
	cmdTemp.Stderr = os.Stderr
	err := cmdTemp.Run()
	if err != nil {
		log.Fatal(err)
	}
	tbNames := strings.Split(b.String(), "\n")
	if len(tbNames) > 1 {
		tablesList += strings.Join(tbNames[1:], "")
	}

	cmd := exec.Command("docker", "exec", "-i", "-u", "mysql", strings.ToLower(projectName)+"-db-1", "mysql", "-u", "root", "-p"+projectConfig["DB_ROOT_PASSWORD"], "-h", "db", "--execute", "SET @@session.unique_checks = 0;SET @@session.foreign_key_checks = 0;SET @@global.innodb_autoinc_lock_mode = 2;SET FOREIGN_KEY_CHECKS=0;"+tablesList+"SET FOREIGN_KEY_CHECKS=1;", "-f", projectConfig["DB_DATABASE"])
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("The database was cleaned successfully")
}
