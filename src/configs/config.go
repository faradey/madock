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

func SetEnvForProject(defVersions versions.ToolsVersions) {
	projectName := paths.GetRunDirName()
	generalConf := GetGeneralConfig()
	envFile := paths.GetExecDirPath() + "/projects/" + projectName + "/env"
	addLine("PHP_VERSION", defVersions.Php)
	addLine("PHP_COMPOSER_VERSION", defVersions.Composer)
	addLine("PHP_TZ", "Europe/Kiev")
	addLine("PHP_XDEBUG_REMOTE_HOST", "host.docker.internal")
	addLine("PHP_MODULE_XDEBUG", "true")
	addLine("PHP_MODULE_IONCUBE", "true")

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

	/* Create nginx default configuration for Magento2 */
	nginxDefFile := paths.GetExecDirPath() + "/docker/nginx/conf/default.conf"
	b, err := os.ReadFile(nginxDefFile)
	if err != nil {
		log.Fatal(err)
	}
	str := string(b)
	str = strings.Replace(str, "{{{NGINX_PORT}}}", generalConf["NGINX_PORT"], -1)
	str = strings.Replace(str, "{{{HOST_NAMES}}}", "loc."+projectName+".com", -1)
	str = strings.Replace(str, "{{{PROJECT_NAME}}}", projectName, -1)
	str = strings.Replace(str, "{{{HOST_NAMES_WEBSITES}}}", "loc."+projectName+".com base;", -1)
	nginxFile := paths.GetExecDirPath() + "/projects/" + projectName + "/docker/nginx/" + projectName + ".conf"
	err = ioutil.WriteFile(nginxFile, []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
	/* END Create nginx default configuration for Magento2 */

	/* Create nginx Dockerfile configuration */
	nginxDefFile = paths.GetExecDirPath() + "/docker/nginx/Dockerfile"
	b, err = os.ReadFile(nginxDefFile)
	if err != nil {
		log.Fatal(err)
	}

	projectsNames := paths.GetDirs(paths.GetExecDirPath() + "/projects")
	copyLines := ""
	for _, name := range projectsNames {
		copyLines += "COPY conf/" + name + ".conf /etc/nginx/sites-enabled/" + name + ".conf\n"
	}

	str = string(b)
	str = strings.Replace(str, "{{{COPY_NGINX_CONF}}}", copyLines, -1)
	usr, err := user.Current()
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
	str = strings.Replace(str, "{{{UID}}}", usr.Uid, -1)
	str = strings.Replace(str, "{{{GUID}}}", usr.Gid, -1)
	paths.MakeDirsByPath(paths.GetExecDirPath() + "/aruntime/nginx")
	err = ioutil.WriteFile(paths.GetExecDirPath()+"/aruntime/nginx/Dockerfile", []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
	/* END Create nginx Dockerfile configuration */

	/* Copy nginx docker-compose configuration */
	nginxDefFile = paths.GetExecDirPath() + "/docker/nginx/docker-compose.yml"
	b, err = os.ReadFile(nginxDefFile)
	if err != nil {
		log.Fatal(err)
	}
	paths.MakeDirsByPath(paths.GetExecDirPath() + "/aruntime/nginx")
	err = ioutil.WriteFile(paths.GetExecDirPath()+"/aruntime/nginx/docker-compose.yml", b, 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
	/* END Create nginx Dockerfile configuration */
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
		fmt.Print("> ")
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
