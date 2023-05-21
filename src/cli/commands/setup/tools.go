package setup

import (
	"bufio"
	"fmt"
	"github.com/faradey/madock/src/cli/fmtc"
	"github.com/faradey/madock/src/paths"
	"log"
	"os"
	"strconv"
	"strings"
)

func Platform() string {
	defVersion := "magento2"
	setTitleAndRecommended("Platform", &defVersion)

	availableVersions := []string{"", "magento2", "pwa"}

	prepareVersions(availableVersions)
	invitation(&defVersion)
	waiterAndProceed(&defVersion, availableVersions)
	return defVersion
}

func Php(defVersion *string) {
	setTitleAndRecommended("PHP", defVersion)

	availableVersions := []string{"Custom", "8.1", "8.0", "7.4", "7.3", "7.2", "7.1", "7.0"}

	prepareVersions(availableVersions)
	invitation(defVersion)
	waiterAndProceed(defVersion, availableVersions)
}

func Db(defVersion *string) {
	setTitleAndRecommended("DB", defVersion)

	availableVersions := []string{"Custom", "10.6", "10.4", "10.3", "10.2", "10.1", "10.0"}

	prepareVersions(availableVersions)
	invitation(defVersion)
	waiterAndProceed(defVersion, availableVersions)
}

func Composer(defVersion *string) {
	setTitleAndRecommended("Composer", defVersion)

	availableVersions := []string{"Custom", "1", "2"}

	prepareVersions(availableVersions)
	invitation(defVersion)
	waiterAndProceed(defVersion, availableVersions)
}

func SearchEngine(defVersion *string) {
	setTitleAndRecommended("Search Engine", defVersion)

	availableVersions := []string{"", "OpenSearch", "Elasticsearch", "Do not use"}

	prepareVersions(availableVersions)
	invitation(defVersion)
	waiterAndProceed(defVersion, availableVersions)
}

func Elastic(defVersion *string) {
	setTitleAndRecommended("Elasticsearch", defVersion)

	availableVersions := []string{"Custom", "8.4.3", "7.17.5", "7.16.3", "7.10.1", "7.9.3", "7.7.1", "7.6.2", "6.8.20", "5.1.2"}

	prepareVersions(availableVersions)
	invitation(defVersion)
	waiterAndProceed(defVersion, availableVersions)
}

func OpenSearch(defVersion *string) {
	setTitleAndRecommended("OpenSearch", defVersion)

	availableVersions := []string{"Custom", "2.5.0", "1.2.0"}

	prepareVersions(availableVersions)
	invitation(defVersion)
	waiterAndProceed(defVersion, availableVersions)
}

func Redis(defVersion *string) {
	setTitleAndRecommended("Redis", defVersion)

	availableVersions := []string{"Custom", "7.0 (Magento version > 2.4.6)", "6.2 (Magento version > 2.4.3-p3)", "6.0 (Magento version <= 2.4.3-p3)", "5.0 (Magento version <= 2.3.2)"}

	prepareVersions(availableVersions)
	invitation(defVersion)
	waiterAndProceed(defVersion, availableVersions)
}

func RabbitMQ(defVersion *string) {
	setTitleAndRecommended("RabbitMQ", defVersion)
	availableVersions := []string{"Custom", "3.9 (Magento version > 2.4.3-p3)", "3.8 (Magento version <= 2.4.3-p3)", "3.7 (Magento version <= 2.3.4)"}

	prepareVersions(availableVersions)
	invitation(defVersion)
	waiterAndProceed(defVersion, availableVersions)
}

func Hosts(projectName string, defVersion *string, projectConfig map[string]string) {
	host := strings.ToLower(projectName + projectConfig["DEFAULT_HOST_FIRST_LEVEL"] + ":base")
	if val, ok := projectConfig["HOSTS"]; ok && val != "" {
		host = val
	}
	fmtc.TitleLn("Hosts")
	fmt.Println("Input format: a.example.com:x_website_code b.example.com:y_website_code")
	fmt.Println("Recommended host: " + host)
	availableVersions := []string{projectName + projectConfig["DEFAULT_HOST_FIRST_LEVEL"] + ":base", "loc." + projectName + ".com:base"}
	prepareVersions(availableVersions)
	fmt.Println("Choose one of the suggested options or enter your hostname")
	fmt.Print("> ")
	selected, _ := Waiter()
	if selected == "" && host != "" {
		*defVersion = host
		fmtc.SuccessLn("Your choice: " + *defVersion)
	} else if selected != "" {
		selectedInt, err := strconv.Atoi(selected)
		if err == nil && len(availableVersions) >= selectedInt {
			*defVersion = availableVersions[selectedInt-1]
		} else {
			*defVersion = selected
		}
		fmtc.SuccessLn("Your choice: " + *defVersion)
	}
}

func NodeJs(defVersion *string) {
	setTitleAndRecommended("NodeJs", defVersion)

	availableVersions := []string{"Custom", "16.20.0", "18.15.0"}

	prepareVersions(availableVersions)
	invitation(defVersion)
	waiterAndProceed(defVersion, availableVersions)
}

func Yarn(defVersion *string) {
	setTitleAndRecommended("Yarn", defVersion)

	availableVersions := []string{"Custom", "1.22.19"}

	prepareVersions(availableVersions)
	invitation(defVersion)
	waiterAndProceed(defVersion, availableVersions)
}

func setTitleAndRecommended(title string, recommended *string) {
	fmt.Println("")
	fmtc.TitleLn(title)
	if *recommended != "" {
		fmt.Println("Recommended version: " + *recommended)
	}
}

func prepareVersions(availableVersions []string) {
	for index, ver := range availableVersions {
		if ver != "" {
			fmt.Println(strconv.Itoa(index) + ") " + ver)
		}
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
	selected, repoAndVersion := Waiter()
	setSelectedVersion(defVersion, availableVersions, selected, repoAndVersion)
}

func Waiter() (selected, repoAndVersion string) {
	buf := bufio.NewReader(os.Stdin)
	sentence, err := buf.ReadBytes('\n')
	if err != nil {
		log.Fatalln(err)
	}
	selected = strings.TrimSpace(string(sentence))
	if selected == "0" {
		fmt.Println("Enter the version")
		fmt.Print("> ")
		buf = bufio.NewReader(os.Stdin)
		sentence, err = buf.ReadBytes('\n')
		if err != nil {
			log.Fatalln(err)
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
	} else if selected != "" && err == nil && len(availableVersions) >= selectedInt {
		*defVersion = availableVersions[selectedInt]
		fmtc.SuccessLn("Your choice: " + *defVersion)
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
	err = os.WriteFile(pathTo, b, 0755)
	if err != nil {
		log.Fatalf("Unable to write file: %v", err)
	}
}
