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
	"strings"
)

type ConfigLines struct {
	Lines   []string
	EnvFile string
}

var dbType = "MariaDB"

func SetEnvForProject(defVersions versions.ToolsVersions) {
	projectName := paths.GetRunDirName()
	generalConf := GetGeneralConfig()
	envFile := paths.GetExecDirPath() + "/projects/" + projectName + "/env"
	config := new(ConfigLines)
	config.EnvFile = envFile
	config.AddLine("PHP_VERSION", defVersions.Php)
	config.AddLine("PHP_COMPOSER_VERSION", defVersions.Composer)
	config.AddLine("PHP_TZ", "Europe/Kiev")
	config.AddLine("PHP_XDEBUG_REMOTE_HOST", "host.docker.internal")
	config.AddLine("PHP_XDEBUG_IDE_KEY", "PHPSTORM")
	config.AddLine("PHP_MODULE_XDEBUG", "false")
	config.AddLine("PHP_MODULE_IONCUBE", "false")
	config.AddLine("PHP_MEMORY_LIMIT", generalConf["PHP_MEMORY_LIMIT"])

	config.AddEmptyLine()

	config.AddLine("DB_VERSION", defVersions.Db)
	config.AddLine("DB_TYPE", dbType)
	config.AddLine("DB_ROOT_PASSWORD", "password")
	config.AddLine("DB_USER", "magento")
	config.AddLine("DB_PASSWORD", "magento")
	config.AddLine("DB_DATABASE", "magento")

	config.AddEmptyLine()

	config.AddLine("ELASTICSEARCH_ENABLE", generalConf["ELASTICSEARCH_ENABLE"])
	config.AddLine("ELASTICSEARCH_VERSION", defVersions.Elastic)

	config.AddEmptyLine()

	config.AddLine("REDIS_ENABLE", generalConf["REDIS_ENABLE"])
	config.AddLine("REDIS_VERSION", defVersions.Redis)

	config.AddEmptyLine()

	config.AddLine("RABBITMQ_ENABLE", generalConf["RABBITMQ_ENABLE"])
	config.AddLine("RABBITMQ_VERSION", defVersions.RabbitMQ)

	config.AddEmptyLine()

	config.AddLine("PHPMYADMIN_ENABLE", generalConf["PHPMYADMIN_ENABLE"])
	config.AddLine("PHPMYADMIN_PORT", generalConf["PHPMYADMIN_PORT"])

	config.AddEmptyLine()

	/*usr, err := user.Current()
	if err == nil {
		AddLine("UID", usr.Uid)
		AddLine("GUID", usr.Gid)
	} else {
		log.Fatal(err)
	}*/

	config.SaveLines()
}

func (t *ConfigLines) AddLine(name, value string) {
	t.Lines = append(t.Lines, name+"="+value)
}

func (t *ConfigLines) AddEmptyLine() {
	t.Lines = append(t.Lines, "")
}

func (t *ConfigLines) AddRawLine(value string) {
	t.Lines = append(t.Lines, value)
}

func (t *ConfigLines) SaveLines() {
	err := ioutil.WriteFile(t.EnvFile, []byte(strings.Join(t.Lines, "\n")), 0755)
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

func ConfigMapping(mainConf map[string]string, targetConf map[string]string) {
	if len(targetConf) > 0 && len(mainConf) > 0 {
		for index, val := range mainConf {
			if _, ok := targetConf[index]; !ok {
				targetConf[index] = val
			}
		}
	}
}
