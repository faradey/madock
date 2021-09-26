package configs

import (
	"bufio"
	"fmt"
	"github.com/faradey/madock/src/cli/fmtc"
	"github.com/faradey/madock/src/paths"
	"github.com/faradey/madock/src/versions"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"strings"
)

var lines []string
var dbType = "MariaDB"

func SetEnvForProject(projectName string, defVersions versions.ToolsVersions) {
	envFile := paths.GetExecDirPath() + "/projects/" + projectName + "/env"
	addLine("PHP_VERSION", defVersions.Php)
	addLine("PHP_COMPOSER_VERSION", defVersions.Composer)
	addLine("PHP_TZ", "Europe/Kiev")
	addLine("PHP_XDEBUG_REMOTE_HOST", "host.docker.internal")
	addLine("PHP_MODULE_XDEBUG", "true")
	addLine("PHP_MODULE_IONCUBE", "true")

	addLine("DB_VERSION", defVersions.Db)
	addLine("DB_TYPE", dbType)
	addLine("DB_DEBUG_PORT", "13306")
	addLine("DB_ROOT_PASSWORD", "password")
	addLine("DB_USER", "magento")
	addLine("DB_PASSWORD", "magento")
	addLine("DB_DATABASE", "magento")
	addLine("DB_DUMP_FILE", "dump.sql.gz")

	addLine("ELASTICSEARCH_VERSION", defVersions.Elastic)
	usr, err := user.Current()
	if err == nil {
		addLine("UID", usr.Uid)
		addLine("GUID", usr.Gid)
	} else {
		log.Fatal(err)
	}

	saveLines(envFile)
}

func addLine(name, value string) {
	lines = append(lines, name+"="+value)
}

func saveLines(envFile string) {
	err := ioutil.WriteFile(envFile, []byte(strings.Join(lines, "\n")), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
}

func IsHasConfig(projectName string) {
	dir := paths.GetExecDirPath() + "/projects/" + projectName
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.Mkdir(dir, 0755)
		if err != nil {
			log.Fatal(err)
		}
	}

	envFile := dir + "/env"
	if _, err := os.Stat(envFile); !os.IsNotExist(err) {
		fmtc.WarningLn("File env is already exist in project" + projectName)
		fmt.Println("Do you want to continue? (y/N)")
	}
	buf := bufio.NewReader(os.Stdin)
	sentence, err := buf.ReadBytes('\n')
	selected := strings.TrimSpace(string(sentence))
	if err != nil {
		log.Fatal(err)
	} else {
		if selected != "y" {
			log.Fatal("Exit")
		}
	}
}
