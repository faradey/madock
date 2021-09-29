package configs

import (
	"bufio"
	"fmt"
	"github.com/faradey/madock/src/cli/fmtc"
	"github.com/faradey/madock/src/configs/aruntime"
	"github.com/faradey/madock/src/paths"
	"github.com/faradey/madock/src/versions"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

var lines []string
var dbType = "MariaDB"

func SetEnvForProject(defVersions versions.ToolsVersions) {
	projectName := paths.GetRunDirName()
	generalConf := GetGeneralConfig()
	envFile := paths.GetExecDirPath() + "/projects/" + projectName + "/env"
	addLine("PHP_VERSION", defVersions.Php)
	addLine("PHP_COMPOSER_VERSION", defVersions.Composer)
	addLine("PHP_TZ", "Europe/Kiev")
	addLine("PHP_XDEBUG_REMOTE_HOST", "host.docker.internal")
	addLine("PHP_MODULE_XDEBUG", "false")
	addLine("PHP_MODULE_IONCUBE", "false")
	addLine("PHP_MEMORY_LIMIT", generalConf["PHP_MEMORY_LIMIT"])

	addEmptyLine()

	addLine("DB_VERSION", defVersions.Db)
	addLine("DB_TYPE", dbType)
	addLine("DB_DEBUG_PORT", "13306")
	addLine("DB_ROOT_PASSWORD", "password")
	addLine("DB_USER", "magento")
	addLine("DB_PASSWORD", "magento")
	addLine("DB_DATABASE", "magento")
	addLine("DB_DUMP_FILE", "dump.sql.gz")

	addEmptyLine()

	addLine("ELASTICSEARCH_ENABLE", generalConf["ELASTICSEARCH_ENABLE"])
	addLine("ELASTICSEARCH_VERSION", defVersions.Elastic)

	addEmptyLine()

	addLine("REDIS_ENABLE", generalConf["REDIS_ENABLE"])
	addLine("REDIS_VERSION", defVersions.Redis)

	addEmptyLine()

	addLine("RABBITMQ_ENABLE", generalConf["RABBITMQ_ENABLE"])
	addLine("RABBITMQ_VERSION", defVersions.RabbitMQ)

	addEmptyLine()

	addLine("PHPMYADMIN_ENABLE", generalConf["PHPMYADMIN_ENABLE"])
	addLine("PHPMYADMIN_PORT", generalConf["PHPMYADMIN_PORT"])

	addEmptyLine()

	/*usr, err := user.Current()
	if err == nil {
		addLine("UID", usr.Uid)
		addLine("GUID", usr.Gid)
	} else {
		log.Fatal(err)
	}*/

	saveLines(envFile)
}

func CreateNginxConfForProject() {
	projectName := paths.GetRunDirName()
	generalConf := GetGeneralConfig()
	paths.MakeDirsByPath(paths.GetExecDirPath() + "/projects/" + projectName + "/docker/nginx")

	aruntime.CreateProjectConf(projectName, generalConf)
	aruntime.CreateDefaultNginxConf()
	aruntime.CreateDefaultNginxDockerfile()
	aruntime.CreateDefaultNginxDockerCompose()
}

func addLine(name, value string) {
	lines = append(lines, name+"="+value)
}

func addEmptyLine() {
	lines = append(lines, "")
}

func saveLines(envFile string) {
	err := ioutil.WriteFile(envFile, []byte(strings.Join(lines, "\n")), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
}

func IsHasConfig() {
	paths.PrepareDirsForProject()
	projectName := paths.GetRunDirName()
	envFile := paths.GetExecDirPath() + "/projects/" + projectName + "/env"
	if _, err := os.Stat(envFile); !os.IsNotExist(err) {
		fmtc.WarningLn("File env is already exist in project " + projectName)
		fmt.Println("Do you want to continue? (y/N)")
		fmt.Print("> ")

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
}

func IsHasNotConfig() bool {
	envFile := paths.GetExecDirPath() + "/projects/" + paths.GetRunDirName() + "/env"
	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		return true
	}
	return false
}
