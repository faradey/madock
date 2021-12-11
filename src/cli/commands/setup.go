package commands

import (
	"bufio"
	"fmt"
	"github.com/faradey/madock/src/cli/fmtc"
	"github.com/faradey/madock/src/configs"
	"github.com/faradey/madock/src/docker/builder"
	"github.com/faradey/madock/src/paths"
	"github.com/faradey/madock/src/versions"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
)

func Setup() {
	if configs.IsHasConfig() {
		builder.Down()
	}
	fmtc.SuccessLn("Start set up environment")
	projectName := paths.GetRunDirName()

	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		paths.MakeDirsByPath(paths.GetRunDirPath() + "/pub/media")
	}

	envFile := paths.MakeDirsByPath(paths.GetExecDirPath()+"/projects/"+projectName) + "/env.txt"
	var projectConfig map[string]string
	if _, err := os.Stat(envFile); !os.IsNotExist(err) {
		projectConfig = configs.GetProjectConfig(projectName)
	}

	toolsDefVersions := versions.GetVersions()

	setupPhp(&toolsDefVersions.Php)
	setupDB(&toolsDefVersions.Db)
	setupComposer(&toolsDefVersions.Composer)
	setupElastic(&toolsDefVersions.Elastic)
	setupRedis(&toolsDefVersions.Redis)
	setupRabbitMQ(&toolsDefVersions.RabbitMQ)
	setupHosts(&toolsDefVersions.Hosts, projectConfig)

	configs.SetEnvForProject(toolsDefVersions, projectConfig)
	paths.MakeDirsByPath(paths.GetExecDirPath() + "/projects/" + projectName + "/backup/db")

	fmtc.SuccessLn("Finish set up environment")
}

func setupPhp(defVersion *string) {
	setTitleAndRecommended("PHP", defVersion)

	availableVersions := []string{"8.1", "8.0", "7.4", "7.3", "7.2", "7.1", "7.0"}

	for index, ver := range availableVersions {
		fmt.Println(strconv.Itoa(index+1) + ") " + ver)
	}

	invitation(defVersion)
	waiterAndProceed(defVersion, availableVersions)
}

func setupDB(defVersion *string) {
	setTitleAndRecommended("MariaDB", defVersion)

	availableVersions := []string{"10.4", "10.3", "10.2", "10.1", "10.0"}

	for index, ver := range availableVersions {
		fmt.Println(strconv.Itoa(index+1) + ") " + ver)
	}

	invitation(defVersion)
	waiterAndProceed(defVersion, availableVersions)
}

func setupComposer(defVersion *string) {
	setTitleAndRecommended("Composer", defVersion)

	availableVersions := []string{"1", "2"}

	for index, ver := range availableVersions {
		fmt.Println(strconv.Itoa(index+1) + ") " + ver)
	}

	invitation(defVersion)
	waiterAndProceed(defVersion, availableVersions)
}

func setupElastic(defVersion *string) {
	setTitleAndRecommended("Elasticsearch", defVersion)

	availableVersions := []string{"7.10.1", "7.9.3", "7.7.1", "7.6.2", "6.8.20", "5.1.2"}

	for index, ver := range availableVersions {
		fmt.Println(strconv.Itoa(index+1) + ") " + ver)
	}

	invitation(defVersion)
	waiterAndProceed(defVersion, availableVersions)
}

func setupRedis(defVersion *string) {
	setTitleAndRecommended("Redis", defVersion)

	availableVersions := []string{"6.0", "5.0"}

	for index, ver := range availableVersions {
		fmt.Println(strconv.Itoa(index+1) + ") " + ver)
	}

	invitation(defVersion)
	waiterAndProceed(defVersion, availableVersions)
}

func setupRabbitMQ(defVersion *string) {
	setTitleAndRecommended("RabbitMQ", defVersion)
	availableVersions := []string{"3.8", "3.7"}
	prepareVersions(availableVersions)
	invitation(defVersion)
	waiterAndProceed(defVersion, availableVersions)
}

func setupHosts(defVersion *string, projectConfig map[string]string) {
	projectName := paths.GetRunDirName()
	host := projectName + ".loc:base"
	if val, ok := projectConfig["HOSTS"]; ok && val != "" {
		host = val
	}
	fmtc.TitleLn("Hosts")
	fmt.Println("Input format: a.example.com:x_website_code b.example.com:y_website_code")
	fmt.Println("Recommended host: " + host)
	availableVersions := []string{host, "loc." + projectName + ".com"}
	prepareVersions(availableVersions)
	fmt.Println("Choose one of the suggested options or enter your hostname")
	fmt.Print("> ")
	selected := waiter()
	if selected == "" && host != "" {
		*defVersion = host
		fmt.Println("Your choice: " + *defVersion)
	} else if selected != "" {
		selectedInt, err := strconv.Atoi(selected)
		if err == nil && len(availableVersions) >= selectedInt {
			*defVersion = availableVersions[selectedInt-1]
		} else {
			*defVersion = selected
		}
		fmt.Println("Your choice: " + *defVersion)
	}
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

func waiterAndProceed(defVersion *string, availableVersions []string) {
	selected := waiter()
	setSelectedVersion(defVersion, availableVersions, selected)
}

func waiter() string {
	buf := bufio.NewReader(os.Stdin)
	sentence, err := buf.ReadBytes('\n')
	if err != nil {
		log.Fatalln(err)
	}
	selected := strings.TrimSpace(string(sentence))

	return selected
}

func setSelectedVersion(defVersion *string, availableVersions []string, selected string) {
	selectedInt, err := strconv.Atoi(selected)
	if selected == "" && *defVersion != "" {
		fmt.Println("Your choice: " + *defVersion)
	} else if selected != "" && err == nil && len(availableVersions) >= selectedInt {
		*defVersion = availableVersions[selectedInt-1]
		fmt.Println("Your choice: " + *defVersion)
	} else {
		fmtc.WarningLn("Choose one of the options offered")
		waiterAndProceed(defVersion, availableVersions)
	}
}

func copyFile(pathFrom, pathTo string) {
	b, err := os.ReadFile(pathFrom)
	if err != nil {
		log.Fatal(err)
	}
	pathToAsSlice := strings.Split(pathTo, "/")
	paths.MakeDirsByPath(strings.Join(pathToAsSlice[:len(pathToAsSlice)-1], "/"))
	err = ioutil.WriteFile(pathTo, b, 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
}
