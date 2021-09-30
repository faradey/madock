package commands

import (
	"bufio"
	"fmt"
	"github.com/faradey/madock/src/cli/fmtc"
	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/paths"
	"github.com/faradey/madock/src/versions"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"strconv"
	"strings"
)

func Setup() {
	configs.IsHasConfig()
	fmtc.SuccessLn("Start set up environment")
	toolsDefVersions := versions.GetVersions()

	setupPhp(&toolsDefVersions.Php)
	setupDB(&toolsDefVersions.Db)
	setupComposer(&toolsDefVersions.Composer)
	setupElastic(&toolsDefVersions.Elastic)
	setupRedis(&toolsDefVersions.Redis)
	setupRabbitMQ(&toolsDefVersions.RabbitMQ)

	configs.SetEnvForProject(toolsDefVersions)

	createProjectNginxConf(configs.GetProjectConfig())
	createProjectNginxDockerfile()

	fmtc.SuccessLn("Finish set up environment")
}

func setupPhp(defVersion *string) {
	setTitleAndRecommended("PHP", defVersion)

	availableVersions := []string{"8.1", "8.0", "7.4", "7.3", "7.2", "7.1", "7.0"}

	for index, ver := range availableVersions {
		fmt.Println(strconv.Itoa(index+1) + ") " + ver)
	}

	invitation(defVersion)

	waiter(defVersion, availableVersions)
}

func setupDB(defVersion *string) {
	setTitleAndRecommended("MariaDB", defVersion)

	availableVersions := []string{"10.4", "10.3", "10.2", "10.1", "10.0"}

	for index, ver := range availableVersions {
		fmt.Println(strconv.Itoa(index+1) + ") " + ver)
	}

	invitation(defVersion)

	waiter(defVersion, availableVersions)
}

func setupComposer(defVersion *string) {
	setTitleAndRecommended("Composer", defVersion)

	availableVersions := []string{"1", "2"}

	for index, ver := range availableVersions {
		fmt.Println(strconv.Itoa(index+1) + ") " + ver)
	}
	invitation(defVersion)

	waiter(defVersion, availableVersions)
}

func setupElastic(defVersion *string) {
	setTitleAndRecommended("Elasticsearch", defVersion)

	availableVersions := []string{"7.10", "7.9", "7.7", "7.6", "6.8", "5.1"}

	for index, ver := range availableVersions {
		fmt.Println(strconv.Itoa(index+1) + ") " + ver)
	}

	invitation(defVersion)

	waiter(defVersion, availableVersions)
}

func setupRedis(defVersion *string) {
	setTitleAndRecommended("Redis", defVersion)

	availableVersions := []string{"6.0", "5.0"}

	for index, ver := range availableVersions {
		fmt.Println(strconv.Itoa(index+1) + ") " + ver)
	}

	invitation(defVersion)

	waiter(defVersion, availableVersions)
}

func setupRabbitMQ(defVersion *string) {
	setTitleAndRecommended("RabbitMQ", defVersion)
	availableVersions := []string{"3.8", "3.7"}
	prepareVersions(availableVersions)
	invitation(defVersion)
	waiter(defVersion, availableVersions)
}

func prepareVersions(availableVersions []string) {
	for index, ver := range availableVersions {
		fmt.Println(strconv.Itoa(index+1) + ") " + ver)
	}
}

func setTitleAndRecommended(title string, recommended *string) {
	fmtc.TitleLn(title)
	if *recommended != "" {
		fmt.Println("Recommended version: " + *recommended)
	}
}

func invitation(ver *string) {
	if *ver != "" {
		fmt.Println("Enter the item number or press Enter to select the recommended version")
	} else {
		fmt.Println("Enter the item number")
	}

	fmt.Print("> ")
}

func waiter(defVersion *string, availableVersions []string) {
	buf := bufio.NewReader(os.Stdin)
	sentence, err := buf.ReadBytes('\n')
	selected := strings.TrimSpace(string(sentence))
	if err != nil {
		log.Fatalln(err)
	} else {
		selectedInt, err := strconv.Atoi(selected)
		if selected == "" && *defVersion != "" {
			fmt.Println("Your choice: " + *defVersion)
		} else if selected != "" && err == nil && len(availableVersions) >= selectedInt {
			*defVersion = availableVersions[selectedInt-1]
			fmt.Println("Your choice: " + *defVersion)
		} else {
			fmtc.WarningLn("Choose one of the options offered")
			setupElastic(defVersion)
		}
	}
}

func createProjectNginxConf(projectConf map[string]string) {
	projectName := paths.GetRunDirName()
	/* Create nginx default configuration for Magento2 */
	nginxDefFile := paths.GetExecDirPath() + "/docker/nginx/conf/default.conf"
	b, err := os.ReadFile(nginxDefFile)
	if err != nil {
		log.Fatal(err)
	}
	str := string(b)
	str = strings.Replace(str, "{{{NGINX_PORT}}}", projectConf["NGINX_PORT"], -1)
	str = strings.Replace(str, "{{{HOST_NAMES}}}", "loc."+projectName+".com", -1)
	str = strings.Replace(str, "{{{PROJECT_NAME}}}", projectName, -1)
	str = strings.Replace(str, "{{{HOST_NAMES_WEBSITES}}}", "loc."+projectName+".com base;", -1)
	nginxFile := paths.GetExecDirPath() + "/projects/" + projectName + "/docker/nginx/default.conf"
	err = ioutil.WriteFile(nginxFile, []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
	/* END Create nginx default configuration for Magento2 */
}

func createProjectNginxDockerfile() {
	/* Create nginx Dockerfile configuration */
	projectName := paths.GetRunDirName()
	paths.MakeDirsByPath(paths.GetExecDirPath() + "/aruntime/ctx")
	nginxDefFile := paths.GetExecDirPath() + "/docker/nginx/Dockerfile"
	b, err := os.ReadFile(nginxDefFile)
	if err != nil {
		log.Fatal(err)
	}

	str := string(b)
	usr, err := user.Current()
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
	str = strings.Replace(str, "{{{UID}}}", usr.Uid, -1)
	str = strings.Replace(str, "{{{GUID}}}", usr.Gid, -1)
	paths.MakeDirsByPath(paths.GetExecDirPath() + "/projects/" + projectName + "/docker/nginx")
	err = ioutil.WriteFile(paths.GetExecDirPath()+"/projects/"+projectName+"/docker/nginx/Dockerfile", []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
	/* END Create nginx Dockerfile configuration */
}

func createProjectNginxDockerCompose() {
	/* Copy nginx docker-compose configuration */
	nginxDefFile := paths.GetExecDirPath() + "/docker/nginx/docker-compose.yml"
	b, err := os.ReadFile(nginxDefFile)
	if err != nil {
		log.Fatal(err)
	}

	str := string(b)

	projectsDirs := paths.GetDirs(paths.GetExecDirPath() + "/aruntime/projects")

	volumes := ""

	for _, dir := range projectsDirs {
		volumes += "      - ./projects/" + dir + "/src/:/var/www/html/" + dir + "/\n"
		nginxConfFile := paths.GetExecDirPath() + "/projects/" + dir + "/docker/nginx/" + dir + ".conf"
		if _, err := os.Stat(nginxConfFile); os.IsNotExist(err) {
			log.Fatal(err)
		}
		confFileData, err := os.ReadFile(nginxConfFile)
		if err != nil {
			log.Fatal(err)
		}
		err = ioutil.WriteFile(paths.GetExecDirPath()+"/aruntime/ctx/"+dir+".conf", confFileData, 0755)
		if err != nil {
			log.Fatalf("Unable to write file: %v", err)
		}
	}

	str = strings.Replace(str, "{{{VOLUMES}}}", volumes, -1)
	err = ioutil.WriteFile(paths.GetExecDirPath()+"/aruntime/docker-compose.yml", []byte(str), 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
	/* END Create nginx Dockerfile configuration */
}
