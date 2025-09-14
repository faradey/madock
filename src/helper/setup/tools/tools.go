package tools

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/faradey/madock/src/helper/cli/fmtc"
	"github.com/faradey/madock/src/helper/configs"
	"github.com/faradey/madock/src/helper/logger"
	"github.com/faradey/madock/src/helper/paths"
)

func Platform() string {
	defVersion := "magento2"
	setTitleAndRecommended("Platform", &defVersion)

	availableVersions := []string{"", "magento2", "pwa", "custom", "shopify", "shopware", "prestashop"}

	PrepareVersions(availableVersions)
	Invitation(&defVersion)
	WaiterAndProceed(&defVersion, availableVersions)
	return defVersion
}

func Php(defVersion *string) {
	setTitleAndRecommended("PHP", defVersion)

	availableVersions := []string{"Custom", "8.4", "8.3", "8.2", "8.1", "8.0", "7.4"}

	PrepareVersions(availableVersions)
	Invitation(defVersion)
	WaiterAndProceed(defVersion, availableVersions)
}

func Db(defVersion *string) {
	setTitleAndRecommended("DB", defVersion)

	availableVersions := []string{"Custom", "11.4", "11.1", "10.6", "10.4", "10.3", "10.2"}

	PrepareVersions(availableVersions)
	Invitation(defVersion)
	WaiterAndProceed(defVersion, availableVersions)
}

func Composer(defVersion *string) {
	setTitleAndRecommended("Composer", defVersion)

	availableVersions := []string{"Custom", "1", "2"}

	PrepareVersions(availableVersions)
	Invitation(defVersion)
	WaiterAndProceed(defVersion, availableVersions)
}

func SearchEngine(defVersion *string) {
	setTitleAndRecommended("Search Engine", defVersion)

	availableVersions := []string{"", "OpenSearch", "Elasticsearch", "Do not use"}

	PrepareVersions(availableVersions)
	Invitation(defVersion)
	WaiterAndProceed(defVersion, availableVersions)
}

func Elastic(defVersion *string) {
	setTitleAndRecommended("Elasticsearch", defVersion)

	availableVersions := []string{"Custom", "8.17.6", "8.11.14", "8.4.3", "7.17.5", "7.16.3", "7.10.1"}

	PrepareVersions(availableVersions)
	Invitation(defVersion)
	WaiterAndProceed(defVersion, availableVersions)
}

func OpenSearch(defVersion *string) {
	setTitleAndRecommended("OpenSearch", defVersion)

	availableVersions := []string{"Custom", "2.19.0", "2.12.0", "2.5.0", "1.2.0"}

	PrepareVersions(availableVersions)
	Invitation(defVersion)
	WaiterAndProceed(defVersion, availableVersions)
}

func Redis(defVersion *string) {
	setTitleAndRecommended("Redis", defVersion)

	availableVersions := []string{"Custom", "8.0", "7.2", "7.0", "6.2", "6.0", "5.0"}

	PrepareVersions(availableVersions)
	Invitation(defVersion)
	WaiterAndProceed(defVersion, availableVersions)
}

func Valkey(defVersion *string) {
	setTitleAndRecommended("Valkey", defVersion)

	availableVersions := []string{"Custom", "8.1.3"}

	PrepareVersions(availableVersions)
	Invitation(defVersion)
	WaiterAndProceed(defVersion, availableVersions)
}

func RabbitMQ(defVersion *string) {
	setTitleAndRecommended("RabbitMQ", defVersion)
	availableVersions := []string{"Custom", "4.1", "3.13", "3.12", "3.9", "3.8", "3.7"}

	PrepareVersions(availableVersions)
	Invitation(defVersion)
	WaiterAndProceed(defVersion, availableVersions)
}

func Hosts(projectName string, defVersion *string, projectConf map[string]string) {
	host := strings.ToLower(projectName + projectConf["nginx/default_host_first_level"] + ":base")
	hosts := configs.GetHosts(projectConf)
	if len(hosts) > 0 {
		var hostItems []string
		for _, hostItem := range hosts {
			hostItems = append(hostItems, hostItem["name"]+":"+hostItem["code"])
		}
		host = strings.Join(hostItems, " ")
	}

	fmtc.TitleLn("Hosts")
	fmt.Println("Input format: a.example.com:x_website_code b.example.com:y_website_code")
	fmt.Println("Recommended host: " + host)
	*defVersion = host
	availableVersions := []string{"Custom", projectName + projectConf["nginx/default_host_first_level"] + ":base", "loc." + projectName + ".com:base"}
	PrepareVersions(availableVersions)
	Invitation(defVersion)
	WaiterAndProceed(defVersion, availableVersions)
}

func NodeJs(defVersion *string) {
	setTitleAndRecommended("NodeJs", defVersion)

	availableVersions := []string{"Custom", "21.1.0", "20.19.0", "18.15.0", "16.20.0"}

	PrepareVersions(availableVersions)
	Invitation(defVersion)
	WaiterAndProceed(defVersion, availableVersions)
}

func Yarn(defVersion *string) {
	setTitleAndRecommended("Yarn", defVersion)

	availableVersions := []string{"Custom", "3.6.4", "1.22.19"}

	PrepareVersions(availableVersions)
	Invitation(defVersion)
	WaiterAndProceed(defVersion, availableVersions)
}

func setTitleAndRecommended(title string, recommended *string) {
	fmt.Println("")
	fmtc.TitleLn(title)
	if *recommended != "" {
		fmt.Println("Recommended version: " + *recommended)
	}
}

func PrepareVersions(availableVersions []string) {
	for index, ver := range availableVersions {
		if ver != "" {
			fmt.Println(strconv.Itoa(index) + ") " + ver)
		}
	}
}

func Invitation(ver *string) {
	if *ver != "" {
		fmt.Println("Enter the item number or press Enter to select the recommended item")
	} else {
		fmt.Println("Enter the item number")
	}

	fmt.Print("> ")
}

func WaiterAndProceed(defVersion *string, availableVersions []string) {
	selected, repoAndVersion := Waiter()
	setSelectedVersion(defVersion, availableVersions, selected, repoAndVersion)
}

func Waiter() (selected, repoAndVersion string) {
	buf := bufio.NewReader(os.Stdin)
	sentence, err := buf.ReadBytes('\n')
	if err != nil {
		logger.Fatalln(err)
	}
	selected = strings.TrimSpace(string(sentence))
	if selected == "0" {
		fmt.Println("Enter the custom value")
		fmt.Print("> ")
		buf = bufio.NewReader(os.Stdin)
		sentence, err = buf.ReadBytes('\n')
		if err != nil {
			logger.Fatalln(err)
		}
		repoAndVersion = strings.TrimSpace(string(sentence))
	}

	return
}

func setSelectedVersion(defVersion *string, availableVersions []string, selected, repoAndVersion string) {
	selectedInt, err := strconv.Atoi(selected)
	if selected == "" && *defVersion != "" {
		fmtc.SuccessLn("Your choice: " + *defVersion)
	} else if selected == "0" {
		*defVersion = repoAndVersion
		fmtc.SuccessLn("Your choice: " + *defVersion)
	} else if selected != "" && err == nil && len(availableVersions) >= selectedInt {
		version := strings.Split(availableVersions[selectedInt], " ")
		*defVersion = version[0]
		fmtc.SuccessLn("Your choice: " + *defVersion)
	} else {
		fmtc.WarningLn("Choose one of the options offered")
		WaiterAndProceed(defVersion, availableVersions)
	}
}

func copyFile(pathFrom, pathTo string) {
	b, err := os.ReadFile(pathFrom)
	if err != nil {
		logger.Fatal(err)
	}
	pathToAsSlice := strings.Split(pathTo, "/")
	paths.MakeDirsByPath(strings.Join(pathToAsSlice[:len(pathToAsSlice)-1], "/"))
	err = os.WriteFile(pathTo, b, 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
}
